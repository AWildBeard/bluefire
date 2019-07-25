# BlueFire: a Command and Control tool that utilizes the Bluetooth Low Energy Protocol

![bluefire](https://byte.farm/content/images/size/w2000/2019/07/bluefire.png)

This project consists of several working components. 
1) The first is the Windows Attack component that is classified as a server. This component uses Windows Platform Libraries to host a BLE GATT server that executes commands passed to it via attribute writes. The Windows Platform component exfiltrates data via GATT attributes aswell.
2) The Second component is the Linux Component. This component is designed to be used with hardware implants on a victims local area network.
3) The Third component is the Client. The client is the initialization point for attackers to view the host machines that have been compromised, selectively connect to them, and execute commands and perform most functions of remote access toolkits.

Due to the nature of BlueTooth, BlueFire will almost always involve a physical penetration testing component to establish a foothold. After a foothold has been established, BlueFire communicates over the not very well monitored Bluetooth Low Energy protocol.

This tool allows red teams to test a corporations ability to detect new and emerging threats, and their ability to counter these threats.