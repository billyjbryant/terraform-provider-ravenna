# Note: request_type_id is write-only — the Ravenna API does not return
# it as a scalar, so an imported status shows it as null.
terraform import ravenna_ticket_status.waiting_on_vendor s_01HXYZ
