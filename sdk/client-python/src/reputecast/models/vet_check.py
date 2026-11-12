from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="VetCheck")



@_attrs_define
class VetCheck:
    """ 
        Attributes:
            key (str):
            result (str):
            detail (str | Unset):
            left (int | Unset):
            need (int | Unset):
            text (str | Unset):
     """

    key: str
    result: str
    detail: str | Unset = UNSET
    left: int | Unset = UNSET
    need: int | Unset = UNSET
    text: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        result = self.result

        detail = self.detail

        left = self.left

        need = self.need

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "result": result,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail
        if left is not UNSET:
            field_dict["left"] = left
        if need is not UNSET:
            field_dict["need"] = need
        if text is not UNSET:
            field_dict["text"] = text

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        result = d.pop("result")

        detail = d.pop("detail", UNSET)

        left = d.pop("left", UNSET)

        need = d.pop("need", UNSET)

        text = d.pop("text", UNSET)

        vet_check = cls(
            key=key,
            result=result,
            detail=detail,
            left=left,
            need=need,
            text=text,
        )

        return vet_check

