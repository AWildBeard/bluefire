using System;
using System.Threading;
using Windows.Devices.Bluetooth.GenericAttributeProfile;
using Windows.Storage.Streams;
using System.Threading.Tasks;
using Windows.Devices.Bluetooth;


namespace WindowsServer
{
    public class Lib
    {
        private static bool peripheralSupported = true;

        private static Command cmd = null;
        
        // Setting up Device
        private GattServiceProviderAdvertisingParameters advParameters = null;

        private GattServiceProvider serviceProvider; // main service
        private GattServiceProviderResult MainService = null;

        private GattLocalCharacteristic readCharacteristic; // STDOUT
        private GattLocalCharacteristic writeCharacteristic; // STDIN
        private GattLocalCharacteristicResult characteristicResult = null;

        private static bool indicate = Constants.gattReadParameters.CharacteristicProperties.HasFlag(GattCharacteristicProperties.Indicate);
        private bool serviceStarted = true;
        public bool hasStarted { get; private set; } = false; // server status
        private bool setIndicate { get; set; } = false;


        /// <summary>
        /// Constructor, initiate BLELib class
        /// </summary>
        /// <param name="main_cmd"></param>
      public void startService(Command main_cmd)
        {
            cmd = main_cmd;
            MainAsync().Wait();
        }

        /// <summary>
        /// Start up the bl;uetooth listeners, set UUIDs/ble-specific information.
        /// </summary>
        /// <returns></returns>
        private async Task MainAsync()
        {
            // Check if adapter supports peripheral and bluetooth low energy
            peripheralSupported = await CheckPeripheralRoleSupportAsync();
            if (!peripheralSupported)
            {
                Environment.Exit(1);
            }


            if (serviceProvider == null)
            {
                ServiceProviderInitAsync().Wait();
                if (serviceStarted)
                {
                    // Advertising server as connectable and discoverable.
                    advParameters = new GattServiceProviderAdvertisingParameters
                    {
                        // IsConnectable determines whether a call to publish will attempt to start advertising and 
                        // put the service UUID in the ADV packet (best effort)
                        IsConnectable = peripheralSupported,
                        IsDiscoverable = true
                    };

                    // Start server
                    serviceProvider.StartAdvertising(advParameters);


                    this.hasStarted = true;
                    if (this.setIndicate == true)
                    {
                        this.setIndicate = false;
                        Indicate();
                    }
                }
            }
            else
            {
                // Stops advertising support
                serviceProvider.StopAdvertising();
                serviceProvider = null;
            }
        }

        /// <summary>
        /// Check that the service has started, start up the characteristics required for the BLE Service
        /// </summary>
        /// <returns></returns>
        private async Task ServiceProviderInitAsync()
        {
            // Initialize and starting a custom GATT Service 
            MainService = await GattServiceProvider.CreateAsync(Constants.serviceProviderUuid);
            if (MainService.Error == BluetoothError.Success) { serviceProvider = MainService.ServiceProvider; }
            else
            {
                // An error occurred.
                serviceStarted = false;
            }

            characteristicResult = await serviceProvider.Service.CreateCharacteristicAsync(Constants.ReadCharacteristicUuid, Constants.gattReadParameters);
            if (characteristicResult.Error != BluetoothError.Success)
            {
                // An error occurred.
                serviceStarted = false;
            }
            readCharacteristic = characteristicResult.Characteristic;
            readCharacteristic.ReadRequested += ReadCharacteristic_ReadRequested;


            characteristicResult = await serviceProvider.Service.CreateCharacteristicAsync(Constants.WriteCharacteristicUuid, Constants.gattWriteParameters);
            if (characteristicResult.Error != BluetoothError.Success)
            {
                // An error occurred.
                serviceStarted = false;
            }
            writeCharacteristic = characteristicResult.Characteristic;
            writeCharacteristic.WriteRequested += WriteCharacteristic_WriteRequested;
        }

        /// <summary>
        /// Method for handling stdin on the Write Characteristic UUID
        /// </summary>
        /// <param name="sender"></param>
        /// <param name="args"></param>
        private async void ReadCharacteristic_ReadRequested(GattLocalCharacteristic sender, GattReadRequestedEventArgs args)
        {
            var deferral = args.GetDeferral();
            var writer = new DataWriter();

            // Buffer to fill cmd output. 
            byte[] cmdCharacter = new byte[1];

            //set the last time the client requested a read - used for indicate

            // Checks if buffer captured data.
            if (cmd.getQueueLength() > 0)
            {
                cmd.setLastReadTime();
                if (cmd.getQueueLength() < 510)
                {
                    cmdCharacter = cmd.getQueuedData(cmd.getQueueLength());
                }
                else
                {
                    cmdCharacter = cmd.getQueuedData(510);
                }
                writer.WriteBytes(cmdCharacter);
                var request = await args.GetRequestAsync();
                request.RespondWithValue(writer.DetachBuffer());
            }
            else
            {
                var empty = new DataWriter();
                empty.WriteByte(0);
                var request = await args.GetRequestAsync();
                request.RespondWithValue(empty.DetachBuffer());
            }

            // Deferal ensures the await task is done. 
            deferral.Complete();
        }

        /// <summary>
        /// handles Reads on the STDOUT characteristic.  If the command thread is done, restart command thread.
        /// </summary>
        /// <param name="sender"></param>
        /// <param name="args"></param>
        private async void WriteCharacteristic_WriteRequested(GattLocalCharacteristic sender, GattWriteRequestedEventArgs args)
        {
            var deferral = args.GetDeferral();

            var request = await args.GetRequestAsync();
            var reader = DataReader.FromBuffer(request.Value);

            //string data = reader.ReadString(reader.UnconsumedBufferLength);
            byte[] data = new byte[reader.UnconsumedBufferLength];
            reader.ReadBytes(data);
            if (cmd.getCmdRunning() == true)
            {
                cmd.writeStdin(data);
                Indicate();
            }
            else
            {
                cmd.startCommand();
                Indicate();
            }

            if (request.Option == GattWriteOption.WriteWithResponse)
            {
                request.Respond();
            }

            deferral.Complete();

        }

        /// <summary>
        /// this method starts up an indicate thread that tells the client that there is more data to grab.
        /// </summary>
        public void Indicate()
        {
            if (hasStarted == true) new Thread(IndicateCheck).Start();
            else setIndicate = true;
        }

        /// <summary>
        /// this is the indicate thread that runs asynchronously, called by Indicate()
        /// </summary>
        private void IndicateCheck()
        {
            var writer = new DataWriter();
            writer.WriteString("Indicate");
            if (indicate)
            {
                readCharacteristic.NotifyValueAsync(writer.DetachBuffer());
            }
        }

        /// <summary>
        /// Make sure that ble is supported on this device.
        /// </summary>
        /// <returns></returns>
        private async Task<bool> CheckPeripheralRoleSupportAsync()
        {
            var localAdapter = await BluetoothAdapter.GetDefaultAsync();

            if (localAdapter != null)
            {
                if (localAdapter.IsLowEnergySupported == false)
                {
                    serviceStarted = false;
                }
                return localAdapter.IsPeripheralRoleSupported;
            }
            else
            {
                // Bluetooth is not turned on.
                return false;
            }
        }
    }
}
