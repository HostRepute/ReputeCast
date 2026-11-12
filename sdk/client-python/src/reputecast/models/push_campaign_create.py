from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="PushCampaignCreate")



@_attrs_define
class PushCampaignCreate:
    """ 
        Attributes:
            name (str):
            link (str | Unset):
            message (str | Unset):
            site_ids (list[str] | Unset):
            title (str | Unset):
     """

    name: str
    link: str | Unset = UNSET
    message: str | Unset = UNSET
    site_ids: list[str] | Unset = UNSET
    title: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        link = self.link

        message = self.message

        site_ids: list[str] | Unset = UNSET
        if not isinstance(self.site_ids, Unset):
            site_ids = self.site_ids



        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })
        if link is not UNSET:
            field_dict["link"] = link
        if message is not UNSET:
            field_dict["message"] = message
        if site_ids is not UNSET:
            field_dict["siteIds"] = site_ids
        if title is not UNSET:
            field_dict["title"] = title

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        link = d.pop("link", UNSET)

        message = d.pop("message", UNSET)

        site_ids = cast(list[str], d.pop("siteIds", UNSET))


        title = d.pop("title", UNSET)

        push_campaign_create = cls(
            name=name,
            link=link,
            message=message,
            site_ids=site_ids,
            title=title,
        )

        return push_campaign_create

