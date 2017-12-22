package hipchat

var ApprovalRequiredTempl = `Approval required!
  %s
  To vote for change type '%s approve %s'
  To reject it: '%s reject %s'
    Votes: %d/%d
    Delta: %s
    Identifier: %s
    Provider: %s`

var VoteReceivedTempl = `Vote received
  Waiting for remaining votes!
    Votes: %d/%d
    Delta: %s
    Identifier: %s`

var ChangeRejectedTempl = `Change rejected
  Change was rejected.
    Status: %s
    Votes: %d/%d
    Delta: %s
    Identifier: %s`

var UpdateApprovedTempl = `Update approved!
  All approvals received, thanks for voting!
    Votes: %d/%d
    Delta: %s
    Identifier: %s`
