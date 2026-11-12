from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.approval_decision_view_decision import ApprovalDecisionViewDecision
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.approval_person import ApprovalPerson





T = TypeVar("T", bound="ApprovalDecisionView")



@_attrs_define
class ApprovalDecisionView:
    """ 
        Attributes:
            approver (ApprovalPerson):
            at (datetime.datetime):
            comment (str):
            decision (ApprovalDecisionViewDecision):
     """

    approver: ApprovalPerson
    at: datetime.datetime
    comment: str
    decision: ApprovalDecisionViewDecision





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        approver = self.approver.to_dict()

        at = self.at.isoformat()

        comment = self.comment

        decision = self.decision.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approver": approver,
            "at": at,
            "comment": comment,
            "decision": decision,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        d = dict(src_dict)
        approver = ApprovalPerson.from_dict(d.pop("approver"))




        at = datetime.datetime.fromisoformat(d.pop("at"))




        comment = d.pop("comment")

        decision = ApprovalDecisionViewDecision(d.pop("decision"))




        approval_decision_view = cls(
            approver=approver,
            at=at,
            comment=comment,
            decision=decision,
        )

        return approval_decision_view

