from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsiteOffFacts")



@_attrs_define
class WebsiteOffFacts:
    """ 
        Attributes:
            paid_members (int):
            public_pages (int):
            shop (bool):
            web_only (int):
     """

    paid_members: int
    public_pages: int
    shop: bool
    web_only: int





    def to_dict(self) -> dict[str, Any]:
        paid_members = self.paid_members

        public_pages = self.public_pages

        shop = self.shop

        web_only = self.web_only


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "paidMembers": paid_members,
            "publicPages": public_pages,
            "shop": shop,
            "webOnly": web_only,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        paid_members = d.pop("paidMembers")

        public_pages = d.pop("publicPages")

        shop = d.pop("shop")

        web_only = d.pop("webOnly")

        website_off_facts = cls(
            paid_members=paid_members,
            public_pages=public_pages,
            shop=shop,
            web_only=web_only,
        )

        return website_off_facts

