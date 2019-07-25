using System;
using System.Threading;

namespace WindowsServer
{
    public class Program
    {

        static Lib bleserver = new Lib();
        static Command ShellService = new Command(bleserver);


        public static void Main(String[] args)
        {
            // Start GATT server
            bleserver.startService(ShellService);

            //spin lock to wait until the thread has started
            while (bleserver.hasStarted == false)
            {
                
            }

            ///Prop open the application, make sure it doesn't unexpectedly close.
            while(bleserver.hasStarted == true)
            {
                Thread.Sleep(500);
            }
        }
    }
}
