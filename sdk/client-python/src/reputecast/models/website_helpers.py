from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="WebsiteHelpers")



@_attrs_define
class WebsiteHelpers:
    """ 
        Attributes:
            manage (list[str]):
            publish (list[str]):
     """

    manage: list[str]
    publish: list[str]





    def to_dict(self) -> dict[str, Any]:
        manage = self.manage



        publish = self.publish




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "manage": manage,
            "publish": publish,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        manage = cast(list[str], d.pop("manage"))


        publish = cast(list[str], d.pop("publish"))


        website_helpers = cls(
            manage=manage,
            publish=publish,
        )

        return website_helpers

