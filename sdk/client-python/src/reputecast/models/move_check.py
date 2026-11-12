from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.move_check_result import MoveCheckResult






T = TypeVar("T", bound="MoveCheck")



@_attrs_define
class MoveCheck:
    """ 
        Attributes:
            key (str):
            result (MoveCheckResult):
            text (str):
     """

    key: str
    result: MoveCheckResult
    text: str





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        result = self.result.value

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "result": result,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        result = MoveCheckResult(d.pop("result"))




        text = d.pop("text")

        move_check = cls(
            key=key,
            result=result,
            text=text,
        )

        return move_check

