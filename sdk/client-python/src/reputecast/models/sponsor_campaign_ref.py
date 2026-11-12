from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SponsorCampaignRef")



@_attrs_define
class SponsorCampaignRef:
    """ 
        Attributes:
            by_hand (bool):
            id (str):
            name (str):
            status (str):
            at (datetime.datetime | Unset):
            author (str | Unset):
     """

    by_hand: bool
    id: str
    name: str
    status: str
    at: datetime.datetime | Unset = UNSET
    author: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        by_hand = self.by_hand

        id = self.id

        name = self.name

        status = self.status

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        author = self.author


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byHand": by_hand,
            "id": id,
            "name": name,
            "status": status,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if author is not UNSET:
            field_dict["author"] = author

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        by_hand = d.pop("byHand")

        id = d.pop("id")

        name = d.pop("name")

        status = d.pop("status")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        author = d.pop("author", UNSET)

        sponsor_campaign_ref = cls(
            by_hand=by_hand,
            id=id,
            name=name,
            status=status,
            at=at,
            author=author,
        )

        return sponsor_campaign_ref

