from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.sponsor_request_item import SponsorRequestItem





T = TypeVar("T", bound="SponsorRequestList")



@_attrs_define
class SponsorRequestList:
    """ 
        Attributes:
            can_manage (bool):
            helpers (list[str]):
            items (list[SponsorRequestItem]):
            page_url (str):
            self_serve (bool):
     """

    can_manage: bool
    helpers: list[str]
    items: list[SponsorRequestItem]
    page_url: str
    self_serve: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_request_item import SponsorRequestItem # noqa: PLC0415
        can_manage = self.can_manage

        helpers = self.helpers



        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        page_url = self.page_url

        self_serve = self.self_serve


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "helpers": helpers,
            "items": items,
            "pageUrl": page_url,
            "selfServe": self_serve,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_request_item import SponsorRequestItem # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        helpers = cast(list[str], d.pop("helpers"))


        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = SponsorRequestItem.from_dict(items_item_data)



            items.append(items_item)


        page_url = d.pop("pageUrl")

        self_serve = d.pop("selfServe")

        sponsor_request_list = cls(
            can_manage=can_manage,
            helpers=helpers,
            items=items,
            page_url=page_url,
            self_serve=self_serve,
        )

        return sponsor_request_list

