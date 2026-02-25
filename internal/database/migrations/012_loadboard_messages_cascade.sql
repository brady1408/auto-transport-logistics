-- +goose Up
ALTER TABLE loadboard_messages
    DROP CONSTRAINT loadboard_messages_claim_id_fkey,
    ADD CONSTRAINT loadboard_messages_claim_id_fkey
        FOREIGN KEY (claim_id) REFERENCES loadboard_claims(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE loadboard_messages
    DROP CONSTRAINT loadboard_messages_claim_id_fkey,
    ADD CONSTRAINT loadboard_messages_claim_id_fkey
        FOREIGN KEY (claim_id) REFERENCES loadboard_claims(id);
