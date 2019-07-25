using System;
using Windows.Devices.Bluetooth.GenericAttributeProfile;


namespace WindowsServer
{

    public class Constants
    {
        // Initializes custom local parameters w/ properties,
        // protection levels as well as common descriptors like User Description.
        public static GattLocalCharacteristicParameters gattWriteParameters = new GattLocalCharacteristicParameters
        {
            CharacteristicProperties = GattCharacteristicProperties.Write |
                                       GattCharacteristicProperties.WriteWithoutResponse,
            WriteProtectionLevel = GattProtectionLevel.Plain,
            UserDescription = "Write Characteristic"
        };

        public static GattLocalCharacteristicParameters gattReadParameters = new GattLocalCharacteristicParameters
        {
            CharacteristicProperties = GattCharacteristicProperties.Read |
                                       GattCharacteristicProperties.Indicate,
            WriteProtectionLevel = GattProtectionLevel.Plain,
            ReadProtectionLevel = GattProtectionLevel.Plain,
            UserDescription = "Read Characteristic"
        };

        public static readonly Guid WriteCharacteristicUuid = Guid.Parse("10a47006-0002-4c30-a9b7-ca7d92240018");
        public static readonly Guid ReadCharacteristicUuid = Guid.Parse("10a47006-0003-4c30-a9b7-ca7d92240018");
        public static readonly Guid serviceProviderUuid = Guid.Parse("10a47006-0001-4c30-a9b7-ca7d92240018");
    };
}
