from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PushPromptRule")



@_attrs_define
class PushPromptRule:
    """ 
        Attributes:
            n (int):
            on (bool):
     """

    n: int
    on: bool





    def to_dict(self) -> dict[str, Any]:
        n = self.n

        on = self.on


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "n": n,
            "on": on,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        n = d.pop("n")

        on = d.pop("on")

        push_prompt_rule = cls(
            n=n,
            on=on,
        )

        return push_prompt_rule

