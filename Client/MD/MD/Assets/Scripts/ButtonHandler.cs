using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using UnityEngine.UI;

public class ButtonHandler : MonoBehaviour
{
    public Button p;
    public void Press(int whichPad)
    {
        Debug.Log("Pressed pad:" + whichPad);
    }
    public Slider slide;
    public static int atWhichOne;
    public void GetSliderPos()
    {
        atWhichOne = (int)slide.value;
    }
}
