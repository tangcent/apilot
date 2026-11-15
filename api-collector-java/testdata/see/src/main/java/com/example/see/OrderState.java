package com.example.see;

/**
 * State of an order.
 */
public enum OrderState {
    /** Waiting for payment. */
    PENDING,
    /**
     * Paid.
     *
     * Ships within a day.
     */
    PAID,
    CLOSED
}
