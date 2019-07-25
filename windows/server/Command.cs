using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Threading;

namespace WindowsServer
{
    public class Command
    {
        //declare threads
        private Thread StdoutThread;
        private Thread ShellThread;
        private Thread StderrThread;
        private Thread queueThread;

        public  Stream output { get; private set; }
        private  bool isCmdRunning = false;
        public  Timer readTicker;
        public  TimeSpan ts;
        private  StreamWriter stdin;
        private  StreamReader stdout;
        private StreamReader stderr;
        private Stream StdinData;

        private  Queue<byte> byteQueue = new Queue<byte>();
        private int queueLength;
        private DateTime lastTime = DateTime.MinValue;
        private Lib lib;

        static object DateTimeLock = new object();
        static object ByteQueueLock = new object();
        static object CommandRunningLock = new object();
        static object queueLengthLock = new object();

        /// <summary>
        /// constructor: initialize one library, run start method.
        /// </summary>
        /// <param name="blelib"></param>
        public Command(Lib blelib)
        {
            this.lib = blelib;
            this.startCommand();
        }

        /// <summary>
        /// this is a thread that asynchronously checks for data to send to the stdout characteristic. 
        /// It does this using the queue intermediary.
        /// </summary>
        private void StdoutWatcher()
        {
            try
            {
                byte[] data;
                //implementing a spin-lock to make sure that powershell is actually running
                while (getCmdRunning() == false)
                {
                    Thread.Sleep(200);
                }
                //we'll see how this looks for performance
                while (getCmdRunning() == true)
                {
                    //create new stream
                    //block until data is read.
                    int data_write = stdout.BaseStream.ReadByte();
                    if (getCmdRunning() && data_write != -1)
                    {
                        writeByte((byte)data_write);
                        addQueueLength();
                    }
                }

            }
            catch(ThreadAbortException)
            {
                
            }
        }

        /// <summary>
        /// this thread is identical to stdoutWatcher, except it checks for the standard error pipe instead of stdout.
        /// </summary>
        private void StderrWatcher()
        {
            try
            {
                byte[] data;
                //implementing a spin-lock to make sure that powershell is actually running
                while (getCmdRunning() == false)
                {
                    Thread.Sleep(200);
                }
                //we'll see how this looks for performance
                while (getCmdRunning() == true)
                {
                    //create new stream
                    //block until data is read.
                    byte data_write = (byte)stderr.BaseStream.ReadByte();
                    writeByte(data_write);
                    addQueueLength();

                }

            }
            catch (ThreadAbortException)
            {

            
            }
        }

        /// <summary>
        /// lock wrapper for the queueLength variable.  Makes the Queue Implementation thread-safe.
        /// </summary>
        /// <param name="length"></param>
        private void setQueueLength(int length)
        {
            lock(queueLengthLock)
            {
                queueLength = length;
            }
        }

        /// <summary>
        /// lock wrapper for adding +1 to queuelength.  Added for ease of use, and helps make queue
        /// implementation thread safe.
        /// </summary>
        private void addQueueLength()
        {
            lock (queueLengthLock)
            {
                queueLength++;
            }
        }
        /// <summary>
        /// lock wrapper for returning queuelength.  Helps make queue implementation thread safe.
        /// </summary>
        /// <returns></returns>
        public int getQueueLength()
        {
            lock(queueLengthLock)
            {
                return queueLength;
            }
        }


        /// <summary>
        /// this thread watches the queue, and as data comes in, checks the last known data read from the client,
        /// and if longer than 250 ms, and there is data sitting in the queue, go ahead and call the ble library
        /// to send an indicate to the client so that it can grab the updated information.
        /// </summary>
        private void queueWatcher()
        {
            try
            {
                long counter = 0;

                //spinlock
                while (getCmdRunning() == false)
                {
                    Thread.Sleep(200);
                }
                while (getCmdRunning() == true)
                {
                    if ((getQueueLength() != 0))
                    {
                        if((DateTime.Now.CompareTo(getLastReadTime().Add(new TimeSpan(0, 0, 0, 0, 250))) >= 0))
                        {
                            counter++;
                            lib.Indicate();
                        }
                    }
                    Thread.Sleep(250);
                }

            }
            catch(ThreadAbortException)
            {
                
            }
        }



        /// <summary>
        /// This is a thread-safe wrapper that adds data to the byte-queue.
        /// </summary>
        /// <param name="b"></param>
        private void writeByte(byte b)
        {
            lock (ByteQueueLock)
            {
                byteQueue.Enqueue(b);
            }
        }
        
        /// <summary>
        /// this is a thread-safe wrapper method to get {length} bytes from the queue, if such data exists.
        /// </summary>
        /// <param name="length"></param>
        /// <returns></returns>
        public byte[] getQueuedData(int length)
        {
            lock (ByteQueueLock)
            {
                //setDate();
                if (length <= getQueueLength() && length > 0)
                {
                    byte[] tempQueue = new byte[length];
                    for (int i = 0; i < length; i++)
                    {
                        tempQueue[i] = byteQueue.Dequeue();
                    }
                    setQueueLength(getQueueLength() - length);
                    return tempQueue;
                }
                else
                {
                    return null;
                }
            }        
        }

        /// <summary>
        /// this is a thread safe wrapper method that returns the powershell session's current status:
        /// running: true
        /// Stopped: false
        /// </summary>
        /// <returns></returns>
        public bool getCmdRunning()
        {
            lock(CommandRunningLock)
            {
                return isCmdRunning;
            }
        }
        /// <summary>
        /// thread-safe method used by the powershell thread to set it's status.
        /// </summary>
        /// <param name="value"></param>
        private void setCmdRunning(bool value)
        {
            lock(CommandRunningLock)
            {
                isCmdRunning = value;
            }
        }

        /// <summary>
        /// return a thread-safe version of the last time that the client has read data.
        /// </summary>
        /// <returns></returns>
        private DateTime getLastReadTime()
        {
            lock (DateTimeLock)
            {
                return this.lastTime;
            }
        }
        /// <summary>
        /// used by BLELib, sets the last time that the client has read data from the server.
        /// </summary>
        public void setLastReadTime()
        {
            lock (DateTimeLock)
            {
                this.lastTime = DateTime.Now;
            }
        }

        /// <summary>
        /// this is a threaded method that launches the powershell session that is used by the rest of this class.  
        /// It starts a process, redirects stdout/stderr/stdin, and ensures that no window is created by the resulting shell.
        /// This command sets {setCmdRunning()} when it starts and stops.
        /// </summary>
        private void runPowershell()
        {
            using (Process process = new Process())
            {
                process.StartInfo.UseShellExecute = false;
                process.StartInfo.CreateNoWindow = true;
                process.StartInfo.RedirectStandardOutput = true;
                process.StartInfo.RedirectStandardError = true;
                process.StartInfo.RedirectStandardInput = true;

                process.StartInfo.FileName = "powershell.exe";

                process.Start();

                stdin = process.StandardInput;
                stdout = process.StandardOutput;
                stderr = process.StandardError;
                setCmdRunning(true);


                process.WaitForExit();
                setCmdRunning(false);
                stopThreads();
                process.Close();
            }
            setCmdRunning(false);
        }

        /// <summary>
        /// Method used to write a byte-array from the client to the powershell session.
        /// </summary>
        /// <param name="data"></param>
        public void writeStdin(byte[] data)
        {
            StdinData = stdin.BaseStream;
            try
            {
                if (getCmdRunning() == true)
                {
                    for (int i = 0; i < data.Length; i++)
                    {
                        //bluefire sends code 127, or delete on backspace.  Here i'm transposing that to code 8, or 
                        //backspace +the previous delete char.
                        if ((int)data[i] == 127)
                        {
                            
                            char[] backspace_code = "\b ".ToCharArray();
                            /*for(int z = 0; z < backspace_code.Length; z ++)
                            {
                                stdin.BaseStream.WriteByte((byte)backspace_code[i]);
                            }*/
                            stdin.Write("\b ");
                            stdin.BaseStream.WriteByte((byte)8);

                        }
                        else
                        {
                            stdin.BaseStream.WriteByte(data[i]);

                        }
                        stdin.BaseStream.Flush();
                    }

                }
                //If the command has unexpectedly died for some reason, restart the session, and then write the data to stdin.
                //this is one of a few checks to make sure that we retain a shell, even on a crash.
                else
                {
                    stopThreads();
                    startCommand();
                    while(getCmdRunning() == false)
                    {
                        Thread.Sleep(200);
                    }
                    writeStdin(data);
                    
                }
                // Writing data from STDIN and STDOUT

            }
            catch
            {
                //handle an exception: one of a few checks to make sure we retain a shell.
                stopThreads();
                startCommand();
                //spinlock
                while(getCmdRunning() == false)
                {
                    Thread.Sleep(200);
                }
                stdin.Write(data);
            }
        }

        /// <summary>
        /// This makes sure that 3/4 of the threads have safely exited.  This method is called
        /// when it is detected that the powershell.exe session has stopped responding.
        /// </summary>
        private void stopThreads()
        {
            if(StdoutThread != null)
            {
                StdoutThread.Abort();
            }
            if(StderrThread != null)
            {
                StderrThread.Abort();
            }
            if (queueThread != null)
            {
                queueThread.Abort();
            }
            StdoutThread.Join();
            StderrThread.Join();
            queueThread.Join();

        }

        /// <summary>
        /// this method starts all of the individual threads we need to run the shell.
        /// </summary>
        public void startCommand()
        {
            ShellThread = new Thread(runPowershell);
            ShellThread.Start();

            StdoutThread = new Thread(StdoutWatcher);
            StdoutThread.Start();

            StderrThread = new Thread(StderrWatcher);
            StderrThread.Start();

            queueThread = new Thread(queueWatcher);
            queueThread.Start();
            
        }

    }
}

