from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="DigestTestInput")



@_attrs_define
class DigestTestInput:
    """ 
        Attributes:
            to (list[str]):
     """

    to: list[str]





    def to_dict(self) -> dict[str, Any]:
        to = self.to




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "to": to,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        to = cast(list[str], d.pop("to"))


        digest_test_input = cls(
            to=to,
        )

        return digest_test_input

