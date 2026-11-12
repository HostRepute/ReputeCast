from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.removal_item import RemovalItem





T = TypeVar("T", bound="LanguageRemoval")



@_attrs_define
class LanguageRemoval:
    """ 
        Attributes:
            code (str):
            fallback (str):
            has (list[RemovalItem]):
            name (str):
            readers (int):
            refusal (str | Unset):
     """

    code: str
    fallback: str
    has: list[RemovalItem]
    name: str
    readers: int
    refusal: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.removal_item import RemovalItem # noqa: PLC0415
        code = self.code

        fallback = self.fallback

        has = []
        for has_item_data in self.has:
            has_item = has_item_data.to_dict()
            has.append(has_item)



        name = self.name

        readers = self.readers

        refusal = self.refusal


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "fallback": fallback,
            "has": has,
            "name": name,
            "readers": readers,
        })
        if refusal is not UNSET:
            field_dict["refusal"] = refusal

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.removal_item import RemovalItem # noqa: PLC0415
        d = dict(src_dict)
        code = d.pop("code")

        fallback = d.pop("fallback")

        has = []
        _has = d.pop("has")
        for has_item_data in (_has):
            has_item = RemovalItem.from_dict(has_item_data)



            has.append(has_item)


        name = d.pop("name")

        readers = d.pop("readers")

        refusal = d.pop("refusal", UNSET)

        language_removal = cls(
            code=code,
            fallback=fallback,
            has=has,
            name=name,
            readers=readers,
            refusal=refusal,
        )

        return language_removal

