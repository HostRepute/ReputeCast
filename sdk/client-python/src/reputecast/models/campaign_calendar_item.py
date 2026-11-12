from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.campaign_calendar_item_kind import CampaignCalendarItemKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CampaignCalendarItem")



@_attrs_define
class CampaignCalendarItem:
    """ 
        Attributes:
            at (datetime.datetime):
            channel (str):
            id (str):
            kind (CampaignCalendarItemKind):
            name (str):
            sent (int):
            status (str):
            approved (bool | Unset):
            campaign_id (str | Unset):
            digest_id (str | Unset):
            local (bool | Unset):
            local_time (str | Unset):
            recipients (int | Unset):
     """

    at: datetime.datetime
    channel: str
    id: str
    kind: CampaignCalendarItemKind
    name: str
    sent: int
    status: str
    approved: bool | Unset = UNSET
    campaign_id: str | Unset = UNSET
    digest_id: str | Unset = UNSET
    local: bool | Unset = UNSET
    local_time: str | Unset = UNSET
    recipients: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        channel = self.channel

        id = self.id

        kind = self.kind.value

        name = self.name

        sent = self.sent

        status = self.status

        approved = self.approved

        campaign_id = self.campaign_id

        digest_id = self.digest_id

        local = self.local

        local_time = self.local_time

        recipients = self.recipients


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "channel": channel,
            "id": id,
            "kind": kind,
            "name": name,
            "sent": sent,
            "status": status,
        })
        if approved is not UNSET:
            field_dict["approved"] = approved
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id
        if digest_id is not UNSET:
            field_dict["digestId"] = digest_id
        if local is not UNSET:
            field_dict["local"] = local
        if local_time is not UNSET:
            field_dict["localTime"] = local_time
        if recipients is not UNSET:
            field_dict["recipients"] = recipients

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        channel = d.pop("channel")

        id = d.pop("id")

        kind = CampaignCalendarItemKind(d.pop("kind"))




        name = d.pop("name")

        sent = d.pop("sent")

        status = d.pop("status")

        approved = d.pop("approved", UNSET)

        campaign_id = d.pop("campaignId", UNSET)

        digest_id = d.pop("digestId", UNSET)

        local = d.pop("local", UNSET)

        local_time = d.pop("localTime", UNSET)

        recipients = d.pop("recipients", UNSET)

        campaign_calendar_item = cls(
            at=at,
            channel=channel,
            id=id,
            kind=kind,
            name=name,
            sent=sent,
            status=status,
            approved=approved,
            campaign_id=campaign_id,
            digest_id=digest_id,
            local=local,
            local_time=local_time,
            recipients=recipients,
        )

        return campaign_calendar_item

