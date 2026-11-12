from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsiteAccessList")



@_attrs_define
class WebsiteAccessList:
    """ 
        Attributes:
            checked (bool):
            consent (str):
            id (str):
            name (str):
            offered (bool):
            public (bool):
     """

    checked: bool
    consent: str
    id: str
    name: str
    offered: bool
    public: bool





    def to_dict(self) -> dict[str, Any]:
        checked = self.checked

        consent = self.consent

        id = self.id

        name = self.name

        offered = self.offered

        public = self.public


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checked": checked,
            "consent": consent,
            "id": id,
            "name": name,
            "offered": offered,
            "public": public,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        checked = d.pop("checked")

        consent = d.pop("consent")

        id = d.pop("id")

        name = d.pop("name")

        offered = d.pop("offered")

        public = d.pop("public")

        website_access_list = cls(
            checked=checked,
            consent=consent,
            id=id,
            name=name,
            offered=offered,
            public=public,
        )

        return website_access_list

