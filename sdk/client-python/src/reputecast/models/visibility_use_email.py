from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="VisibilityUseEmail")



@_attrs_define
class VisibilityUseEmail:
    """ 
        Attributes:
            blocks (list[str]):
            campaign_id (str):
            name (str):
            paths (list[list[int]]):
            status (str):
            scheduled_at (datetime.datetime | Unset):
     """

    blocks: list[str]
    campaign_id: str
    name: str
    paths: list[list[int]]
    status: str
    scheduled_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        blocks = self.blocks



        campaign_id = self.campaign_id

        name = self.name

        paths = []
        for paths_item_data in self.paths:
            paths_item = paths_item_data


            paths.append(paths_item)



        status = self.status

        scheduled_at: str | Unset = UNSET
        if not isinstance(self.scheduled_at, Unset):
            scheduled_at = self.scheduled_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
            "campaignId": campaign_id,
            "name": name,
            "paths": paths,
            "status": status,
        })
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        blocks = cast(list[str], d.pop("blocks"))


        campaign_id = d.pop("campaignId")

        name = d.pop("name")

        paths = []
        _paths = d.pop("paths")
        for paths_item_data in (_paths):
            paths_item = cast(list[int], paths_item_data)

            paths.append(paths_item)


        status = d.pop("status")

        _scheduled_at = d.pop("scheduledAt", UNSET)
        scheduled_at: datetime.datetime | Unset
        if isinstance(_scheduled_at,  Unset):
            scheduled_at = UNSET
        else:
            scheduled_at = datetime.datetime.fromisoformat(_scheduled_at)




        visibility_use_email = cls(
            blocks=blocks,
            campaign_id=campaign_id,
            name=name,
            paths=paths,
            status=status,
            scheduled_at=scheduled_at,
        )

        return visibility_use_email

