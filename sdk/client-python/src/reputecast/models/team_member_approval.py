from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TeamMemberApproval")



@_attrs_define
class TeamMemberApproval:
    """ 
        Attributes:
            at (datetime.datetime):
            campaign (str):
            status (str): approve, request_changes, reject (given) or the request status (requested)
            campaign_id (str | Unset):
     """

    at: datetime.datetime
    campaign: str
    status: str
    campaign_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        campaign = self.campaign

        status = self.status

        campaign_id = self.campaign_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "campaign": campaign,
            "status": status,
        })
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        campaign = d.pop("campaign")

        status = d.pop("status")

        campaign_id = d.pop("campaignId", UNSET)

        team_member_approval = cls(
            at=at,
            campaign=campaign,
            status=status,
            campaign_id=campaign_id,
        )

        return team_member_approval

