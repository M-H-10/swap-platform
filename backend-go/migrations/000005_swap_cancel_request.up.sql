-- An active swap holds money from both parties, so one party alone must
-- not be able to cancel it. The first party asks (this column), and the
-- swap is cancelled only when the other party confirms.
ALTER TABLE swaps
    ADD COLUMN cancel_requested_by UUID REFERENCES users (id) ON DELETE RESTRICT;