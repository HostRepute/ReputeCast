from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_decision_input_decision import ReferralDecisionInputDecision
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ReferralDecisionInput")



@_attrs_define
class ReferralDecisionInput:
    """ 
        Attributes:
            decision (ReferralDecisionInputDecision):
            ids (list[str]):
            note (str | Unset):
     """

    decision: ReferralDecisionInputDecision
    ids: list[str]
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        decision = self.decision.value

        ids = self.ids



        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "decision": decision,
            "ids": ids,
        })
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        decision = ReferralDecisionInputDecision(d.pop("decision"))




        ids = cast(list[str], d.pop("ids"))


        note = d.pop("note", UNSET)

        referral_decision_input = cls(
            decision=decision,
            ids=ids,
            note=note,
        )

        return referral_decision_input

