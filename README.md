# BlueFire
A Command and Control tool that utilizes the Bluetooth Low Energy Protocol

![bluefire](https://raw.githubusercontent.com/AWildBeard/resources/master/Screenshot%202020-04-24%20at%2016.47.06.png)

This project is the creation of Joshua Niemann, Sara Aladham, and Michael Mitchell. It consists of several working components:
1) The First is the Windows Attack component that is classified as a server. This component uses Windows Platform Libraries to host a BLE GATT server that executes commands passed to it via attribute writes. The Windows Platform component exfiltrates data via GATT attributes aswell.
2) The Second component is the Linux Component. This component is designed to be used with hardware implants on a victims local area network.
3) The Third component is the Client. The client is the initialization point for attackers to view the host machines that have been compromised, selectively connect to them, and execute commands and perform most functions of remote access toolkits.

Due to the nature of BlueTooth, BlueFire will almost always involve a physical penetration testing component to establish a foothold. After a foothold has been established, BlueFire communicates over the not very well monitored Bluetooth Low Energy protocol.

## Authors
1) Sara Aladham
2) Joshua Nieman
3) Michael Mitchell
