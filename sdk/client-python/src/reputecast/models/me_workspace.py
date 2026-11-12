from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="MeWorkspace")



@_attrs_define
class MeWorkspace:
    """ 
        Attributes:
            id (str):
            member_count (int):
            name (str):
            role_key (str):
            role_name (str):
            slug (str):
            subscribers (int):
     """

    id: str
    member_count: int
    name: str
    role_key: str
    role_name: str
    slug: str
    subscribers: int





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        member_count = self.member_count

        name = self.name

        role_key = self.role_key

        role_name = self.role_name

        slug = self.slug

        subscribers = self.subscribers


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "memberCount": member_count,
            "name": name,
            "roleKey": role_key,
            "roleName": role_name,
            "slug": slug,
            "subscribers": subscribers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        member_count = d.pop("memberCount")

        name = d.pop("name")

        role_key = d.pop("roleKey")

        role_name = d.pop("roleName")

        slug = d.pop("slug")

        subscribers = d.pop("subscribers")

        me_workspace = cls(
            id=id,
            member_count=member_count,
            name=name,
            role_key=role_key,
            role_name=role_name,
            slug=slug,
            subscribers=subscribers,
        )

        return me_workspace

