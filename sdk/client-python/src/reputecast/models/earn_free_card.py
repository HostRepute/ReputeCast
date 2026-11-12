from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.earn_list_ref import EarnListRef





T = TypeVar("T", bound="EarnFreeCard")



@_attrs_define
class EarnFreeCard:
    """ 
        Attributes:
            line (str):
            readers (int):
            text (str):
            list_ (EarnListRef | Unset):
     """

    line: str
    readers: int
    text: str
    list_: EarnListRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_list_ref import EarnListRef # noqa: PLC0415
        line = self.line

        readers = self.readers

        text = self.text

        list_: dict[str, Any] | Unset = UNSET
        if not isinstance(self.list_, Unset):
            list_ = self.list_.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "line": line,
            "readers": readers,
            "text": text,
        })
        if list_ is not UNSET:
            field_dict["list"] = list_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_list_ref import EarnListRef # noqa: PLC0415
        d = dict(src_dict)
        line = d.pop("line")

        readers = d.pop("readers")

        text = d.pop("text")

        _list_ = d.pop("list", UNSET)
        list_: EarnListRef | Unset
        if isinstance(_list_,  Unset):
            list_ = UNSET
        else:
            list_ = EarnListRef.from_dict(_list_)




        earn_free_card = cls(
            line=line,
            readers=readers,
            text=text,
            list_=list_,
        )

        return earn_free_card

