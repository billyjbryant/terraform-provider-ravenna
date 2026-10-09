# Note: request_channel_id and triage_channel_id are write-only — the
# Ravenna API never returns them, so an imported channel shows them as
# null. Set them in config only when creating a new channel.
terraform import ravenna_channel.it_helpdesk q_01HXYZ
