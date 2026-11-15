package com.example.recovery

import org.hibernate.validator.constraints.Range

/**
 * Settlement record query.
 */
class SettlementQuery {

    /**
     * Record number.
     */
    var recordNo: String? = null

    /**
     * Generated state: 0 pending, 1 generated.
     */
    @Range(min = Limits.ZERO.toLong() , max = Limits.ONE.toLong(), message = Messages.RANGE_ERROR)
    var generatedState: Int? = null

    /**
     * Operator.
     */
    var operatorId: Long? = null
}
