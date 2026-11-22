package com.example.bytearray;

import java.util.List;
import java.util.Map;
import javax.validation.constraints.NotNull;

public class ByteArrayForm {
    /** Written as one Base64 string. */
    private byte[] data;

    /** The brackets may follow the name. */
    private byte legacy[];

    private byte[][] chunks;
    private List<byte[]> blobs;

    /** Spaces may stand in front of the brackets and between them. */
    private byte [ ] spacedData;
    private byte [ ] [ ] spacedChunks;
    private List<byte []> spacedBlobs;
    private Map<String, byte [ ]> spacedByName;
    private List<byte @NotNull []> annotatedBlobs;

    /** Boxed bytes are numbers, one by one. */
    private Byte[] boxed;
    private List<Byte> numbers;
    private byte single;
    private int[] ints;

    public byte[] getSignature() {
        return null;
    }
}
