from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ApprovalPolicySummary")



@_attrs_define
class ApprovalPolicySummary:
    """ 
        Attributes:
            approvals_required (int):
            approvers (list[str]):
            rules (list[str]):
            median_wait_minutes (int | Unset):
     """

    approvals_required: int
    approvers: list[str]
    rules: list[str]
    median_wait_minutes: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        approvals_required = self.approvals_required

        approvers = self.approvers



        rules = self.rules



        median_wait_minutes = self.median_wait_minutes


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvalsRequired": approvals_required,
            "approvers": approvers,
            "rules": rules,
        })
        if median_wait_minutes is not UNSET:
            field_dict["medianWaitMinutes"] = median_wait_minutes

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        approvals_required = d.pop("approvalsRequired")

        approvers = cast(list[str], d.pop("approvers"))


        rules = cast(list[str], d.pop("rules"))


        median_wait_minutes = d.pop("medianWaitMinutes", UNSET)

        approval_policy_summary = cls(
            approvals_required=approvals_required,
            approvers=approvers,
            rules=rules,
            median_wait_minutes=median_wait_minutes,
        )

        return approval_policy_summary

