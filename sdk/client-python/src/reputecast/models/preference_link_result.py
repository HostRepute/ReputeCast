from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PreferenceLinkResult")



@_attrs_define
class PreferenceLinkResult:
    """ 
        Attributes:
            url (str):
     """

    url: str





    def to_dict(self) -> dict[str, Any]:
        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "url": url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        url = d.pop("url")

        preference_link_result = cls(
            url=url,
        )

        return preference_link_result

