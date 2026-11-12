from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.campaign_follow_up_input_kind import CampaignFollowUpInputKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="CampaignFollowUpInput")



@_attrs_define
class CampaignFollowUpInput:
    """ 
        Attributes:
            kind (CampaignFollowUpInputKind):
            subject (str):
            delay_hours (int | Unset):
     """

    kind: CampaignFollowUpInputKind
    subject: str
    delay_hours: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        subject = self.subject

        delay_hours = self.delay_hours


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "subject": subject,
        })
        if delay_hours is not UNSET:
            field_dict["delayHours"] = delay_hours

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = CampaignFollowUpInputKind(d.pop("kind"))




        subject = d.pop("subject")

        delay_hours = d.pop("delayHours", UNSET)

        campaign_follow_up_input = cls(
            kind=kind,
            subject=subject,
            delay_hours=delay_hours,
        )

        return campaign_follow_up_input

