from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SignupFormList")



@_attrs_define
class SignupFormList:
    """ 
        Attributes:
            description (str):
            id (str):
            name (str):
            public (bool):
            subscribed (int):
     """

    description: str
    id: str
    name: str
    public: bool
    subscribed: int





    def to_dict(self) -> dict[str, Any]:
        description = self.description

        id = self.id

        name = self.name

        public = self.public

        subscribed = self.subscribed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "description": description,
            "id": id,
            "name": name,
            "public": public,
            "subscribed": subscribed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        description = d.pop("description")

        id = d.pop("id")

        name = d.pop("name")

        public = d.pop("public")

        subscribed = d.pop("subscribed")

        signup_form_list = cls(
            description=description,
            id=id,
            name=name,
            public=public,
            subscribed=subscribed,
        )

        return signup_form_list

