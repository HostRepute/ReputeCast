from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.traffic_action_kind import TrafficActionKind
from typing import cast
import datetime






T = TypeVar("T", bound="TrafficAction")



@_attrs_define
class TrafficAction:
    """ 
        Attributes:
            at (datetime.datetime):
            campaign (str):
            campaign_id (str):
            kind (TrafficActionKind):
            text (str):
     """

    at: datetime.datetime
    campaign: str
    campaign_id: str
    kind: TrafficActionKind
    text: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        campaign = self.campaign

        campaign_id = self.campaign_id

        kind = self.kind.value

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "campaign": campaign,
            "campaignId": campaign_id,
            "kind": kind,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        campaign = d.pop("campaign")

        campaign_id = d.pop("campaignId")

        kind = TrafficActionKind(d.pop("kind"))




        text = d.pop("text")

        traffic_action = cls(
            at=at,
            campaign=campaign,
            campaign_id=campaign_id,
            kind=kind,
            text=text,
        )

        return traffic_action

