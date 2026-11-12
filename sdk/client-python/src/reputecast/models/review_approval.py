from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.approval_person import ApprovalPerson
  from ..models.approval_request_view import ApprovalRequestView
  from ..models.approval_rule_match import ApprovalRuleMatch





T = TypeVar("T", bound="ReviewApproval")



@_attrs_define
class ReviewApproval:
    """ 
        Attributes:
            approvers (list[ApprovalPerson]):
            can_bypass (bool):
            can_request (bool):
            emergency_bypass (bool):
            needed (int):
            owners (list[str]):
            policy_rules (list[str]):
            required (bool):
            rules (list[ApprovalRuleMatch]):
            valid (bool):
            median_wait_minutes (int | Unset):
            request (ApprovalRequestView | Unset):
            request_reason (str | Unset):
     """

    approvers: list[ApprovalPerson]
    can_bypass: bool
    can_request: bool
    emergency_bypass: bool
    needed: int
    owners: list[str]
    policy_rules: list[str]
    required: bool
    rules: list[ApprovalRuleMatch]
    valid: bool
    median_wait_minutes: int | Unset = UNSET
    request: ApprovalRequestView | Unset = UNSET
    request_reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        from ..models.approval_request_view import ApprovalRequestView # noqa: PLC0415
        from ..models.approval_rule_match import ApprovalRuleMatch # noqa: PLC0415
        approvers = []
        for approvers_item_data in self.approvers:
            approvers_item = approvers_item_data.to_dict()
            approvers.append(approvers_item)



        can_bypass = self.can_bypass

        can_request = self.can_request

        emergency_bypass = self.emergency_bypass

        needed = self.needed

        owners = self.owners



        policy_rules = self.policy_rules



        required = self.required

        rules = []
        for rules_item_data in self.rules:
            rules_item = rules_item_data.to_dict()
            rules.append(rules_item)



        valid = self.valid

        median_wait_minutes = self.median_wait_minutes

        request: dict[str, Any] | Unset = UNSET
        if not isinstance(self.request, Unset):
            request = self.request.to_dict()

        request_reason = self.request_reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvers": approvers,
            "canBypass": can_bypass,
            "canRequest": can_request,
            "emergencyBypass": emergency_bypass,
            "needed": needed,
            "owners": owners,
            "policyRules": policy_rules,
            "required": required,
            "rules": rules,
            "valid": valid,
        })
        if median_wait_minutes is not UNSET:
            field_dict["medianWaitMinutes"] = median_wait_minutes
        if request is not UNSET:
            field_dict["request"] = request
        if request_reason is not UNSET:
            field_dict["requestReason"] = request_reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        from ..models.approval_request_view import ApprovalRequestView # noqa: PLC0415
        from ..models.approval_rule_match import ApprovalRuleMatch # noqa: PLC0415
        d = dict(src_dict)
        approvers = []
        _approvers = d.pop("approvers")
        for approvers_item_data in (_approvers):
            approvers_item = ApprovalPerson.from_dict(approvers_item_data)



            approvers.append(approvers_item)


        can_bypass = d.pop("canBypass")

        can_request = d.pop("canRequest")

        emergency_bypass = d.pop("emergencyBypass")

        needed = d.pop("needed")

        owners = cast(list[str], d.pop("owners"))


        policy_rules = cast(list[str], d.pop("policyRules"))


        required = d.pop("required")

        rules = []
        _rules = d.pop("rules")
        for rules_item_data in (_rules):
            rules_item = ApprovalRuleMatch.from_dict(rules_item_data)



            rules.append(rules_item)


        valid = d.pop("valid")

        median_wait_minutes = d.pop("medianWaitMinutes", UNSET)

        _request = d.pop("request", UNSET)
        request: ApprovalRequestView | Unset
        if isinstance(_request,  Unset):
            request = UNSET
        else:
            request = ApprovalRequestView.from_dict(_request)




        request_reason = d.pop("requestReason", UNSET)

        review_approval = cls(
            approvers=approvers,
            can_bypass=can_bypass,
            can_request=can_request,
            emergency_bypass=emergency_bypass,
            needed=needed,
            owners=owners,
            policy_rules=policy_rules,
            required=required,
            rules=rules,
            valid=valid,
            median_wait_minutes=median_wait_minutes,
            request=request,
            request_reason=request_reason,
        )

        return review_approval

