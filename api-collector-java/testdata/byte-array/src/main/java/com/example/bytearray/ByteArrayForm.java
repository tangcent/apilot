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

    /** A char[] is written as one string of its characters. */
    private char[] letters;
    private char legacyLetters[];
    private char[][] lines;
    private List<char[]> words;
    private char [ ] spacedLetters;
    private List<char @NotNull []> annotatedWords;
    private Map<String, char [ ]> lettersByName;

    /** Boxed characters are strings, one by one. */
    private Character[] boxedLetters;
    private List<Character> letterList;
    private char letter;

    public byte[] getSignature() {
        return null;
    }

    public char[] getPassword() {
        return null;
    }
}
