from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.approval_client_signoff_info import ApprovalClientSignoffInfo
  from ..models.approval_pending_item import ApprovalPendingItem
  from ..models.approval_person import ApprovalPerson
  from ..models.approval_policy import ApprovalPolicy
  from ..models.approval_policy_matches import ApprovalPolicyMatches





T = TypeVar("T", bound="ApprovalPolicyView")



@_attrs_define
class ApprovalPolicyView:
    """ 
        Attributes:
            approvers (list[ApprovalPerson]):
            bypass_count (int):
            can_manage (bool):
            matches (ApprovalPolicyMatches):
            pending (list[ApprovalPendingItem]):
            policy (ApprovalPolicy):
            client_signoff (ApprovalClientSignoffInfo | Unset):
            reason (str | Unset):
     """

    approvers: list[ApprovalPerson]
    bypass_count: int
    can_manage: bool
    matches: ApprovalPolicyMatches
    pending: list[ApprovalPendingItem]
    policy: ApprovalPolicy
    client_signoff: ApprovalClientSignoffInfo | Unset = UNSET
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_client_signoff_info import ApprovalClientSignoffInfo # noqa: PLC0415
        from ..models.approval_pending_item import ApprovalPendingItem # noqa: PLC0415
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        from ..models.approval_policy import ApprovalPolicy # noqa: PLC0415
        from ..models.approval_policy_matches import ApprovalPolicyMatches # noqa: PLC0415
        approvers = []
        for approvers_item_data in self.approvers:
            approvers_item = approvers_item_data.to_dict()
            approvers.append(approvers_item)



        bypass_count = self.bypass_count

        can_manage = self.can_manage

        matches = self.matches.to_dict()

        pending = []
        for pending_item_data in self.pending:
            pending_item = pending_item_data.to_dict()
            pending.append(pending_item)



        policy = self.policy.to_dict()

        client_signoff: dict[str, Any] | Unset = UNSET
        if not isinstance(self.client_signoff, Unset):
            client_signoff = self.client_signoff.to_dict()

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvers": approvers,
            "bypassCount": bypass_count,
            "canManage": can_manage,
            "matches": matches,
            "pending": pending,
            "policy": policy,
        })
        if client_signoff is not UNSET:
            field_dict["clientSignoff"] = client_signoff
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_client_signoff_info import ApprovalClientSignoffInfo # noqa: PLC0415
        from ..models.approval_pending_item import ApprovalPendingItem # noqa: PLC0415
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        from ..models.approval_policy import ApprovalPolicy # noqa: PLC0415
        from ..models.approval_policy_matches import ApprovalPolicyMatches # noqa: PLC0415
        d = dict(src_dict)
        approvers = []
        _approvers = d.pop("approvers")
        for approvers_item_data in (_approvers):
            approvers_item = ApprovalPerson.from_dict(approvers_item_data)



            approvers.append(approvers_item)


        bypass_count = d.pop("bypassCount")

        can_manage = d.pop("canManage")

        matches = ApprovalPolicyMatches.from_dict(d.pop("matches"))




        pending = []
        _pending = d.pop("pending")
        for pending_item_data in (_pending):
            pending_item = ApprovalPendingItem.from_dict(pending_item_data)



            pending.append(pending_item)


        policy = ApprovalPolicy.from_dict(d.pop("policy"))




        _client_signoff = d.pop("clientSignoff", UNSET)
        client_signoff: ApprovalClientSignoffInfo | Unset
        if isinstance(_client_signoff,  Unset):
            client_signoff = UNSET
        else:
            client_signoff = ApprovalClientSignoffInfo.from_dict(_client_signoff)




        reason = d.pop("reason", UNSET)

        approval_policy_view = cls(
            approvers=approvers,
            bypass_count=bypass_count,
            can_manage=can_manage,
            matches=matches,
            pending=pending,
            policy=policy,
            client_signoff=client_signoff,
            reason=reason,
        )

        return approval_policy_view

