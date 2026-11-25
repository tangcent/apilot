package com.example.nested;

import java.util.List;

public class NestHolder {
    private Detail own;
    private NestHolder.Detail qualified;
    private NestOther.Detail other;
    private List<NestOther.Detail> others;
    private Mid.Deep deep;
    private NestBadge badge;
    private String title;

    public static class Detail {
        private String code;
    }

    public static class NestBadge {
        private String text;
    }

    public static class Mid {
        private Detail detail;
        private Deep deep;

        public static class Deep {
            private int depth;
        }
    }
}
