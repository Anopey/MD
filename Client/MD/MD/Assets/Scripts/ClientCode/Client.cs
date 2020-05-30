using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using System.Net.Sockets;
using System.Net;
using Assets.Scripts.ClientCode;
using System.Text;

public static class Client
{

    public static bool displayNetDebug = true;



    public const int port = 52515;
    public static TcpClient tcpClient { get; private set; }
    private static NetworkStream netStream { get; set; }

    public static void SetClient(TcpClient client)
    {
        if (tcpClient == null)
        {
            tcpClient = client;
        }
        else
        {
            WriteToServer("MD CLOSE");
            tcpClient.Close();
            tcpClient = client;
        }
        netStream = client.GetStream();
        ReadFromServer();
    }

    #region Game 

    public static void ReadFromServer()
    {
        byte[] response = new byte[tcpClient.ReceiveBufferSize];
        netStream.Read(response, 0, (int)tcpClient.ReceiveBufferSize);

        string returnData = Encoding.UTF8.GetString(response);
        Debug.Log("Server Response: " + returnData);
    }


    #endregion

    #region Client-Specific Utilities

    public static bool WriteToServer(string message)
    {
        if (tcpClient == null)
            return false;
        byte[] data = Utils.GetBytes(message);
        if (displayNetDebug)
        {
            Debug.Log("Writing to server: " + message);
        }
        tcpClient.GetStream().Write(data, 0, data.Length);
        return true;
    }


    #endregion

}
