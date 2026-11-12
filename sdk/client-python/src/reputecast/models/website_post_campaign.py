from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WebsitePostCampaign")



@_attrs_define
class WebsitePostCampaign:
    """ 
        Attributes:
            id (str):
            name (str):
            status (str):
            scheduled_at (datetime.datetime | Unset):
            sent_at (datetime.datetime | Unset):
     """

    id: str
    name: str
    status: str
    scheduled_at: datetime.datetime | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        status = self.status

        scheduled_at: str | Unset = UNSET
        if not isinstance(self.scheduled_at, Unset):
            scheduled_at = self.scheduled_at.isoformat()

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "status": status,
        })
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        status = d.pop("status")

        _scheduled_at = d.pop("scheduledAt", UNSET)
        scheduled_at: datetime.datetime | Unset
        if isinstance(_scheduled_at,  Unset):
            scheduled_at = UNSET
        else:
            scheduled_at = datetime.datetime.fromisoformat(_scheduled_at)




        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        website_post_campaign = cls(
            id=id,
            name=name,
            status=status,
            scheduled_at=scheduled_at,
            sent_at=sent_at,
        )

        return website_post_campaign

