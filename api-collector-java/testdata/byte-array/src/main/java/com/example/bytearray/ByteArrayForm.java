package com.example.bytearray;

import java.util.List;

public class ByteArrayForm {
    /** Written as one Base64 string. */
    private byte[] data;

    /** The brackets may follow the name. */
    private byte legacy[];

    private byte[][] chunks;
    private List<byte[]> blobs;

    /** Boxed bytes are numbers, one by one. */
    private Byte[] boxed;
    private List<Byte> numbers;
    private byte single;
    private int[] ints;

    public byte[] getSignature() {
        return null;
    }
}
