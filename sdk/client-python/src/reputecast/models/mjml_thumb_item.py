from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.mjml_thumb_item_kind import MJMLThumbItemKind
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="MJMLThumbItem")



@_attrs_define
class MJMLThumbItem:
    """ 
        Attributes:
            accent (bool | Unset):
            h (int | Unset):
            kind (MJMLThumbItemKind | Unset): What the part stands for; empty in thumbnails stored before kinds
            row (list[MJMLThumbItem] | Unset):
            w (int | Unset):
     """

    accent: bool | Unset = UNSET
    h: int | Unset = UNSET
    kind: MJMLThumbItemKind | Unset = UNSET
    row: list[MJMLThumbItem] | Unset = UNSET
    w: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        accent = self.accent

        h = self.h

        kind: str | Unset = UNSET
        if not isinstance(self.kind, Unset):
            kind = self.kind.value


        row: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.row, Unset):
            row = []
            for row_item_data in self.row:
                row_item = row_item_data.to_dict()
                row.append(row_item)



        w = self.w


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if accent is not UNSET:
            field_dict["accent"] = accent
        if h is not UNSET:
            field_dict["h"] = h
        if kind is not UNSET:
            field_dict["kind"] = kind
        if row is not UNSET:
            field_dict["row"] = row
        if w is not UNSET:
            field_dict["w"] = w

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        accent = d.pop("accent", UNSET)

        h = d.pop("h", UNSET)

        _kind = d.pop("kind", UNSET)
        kind: MJMLThumbItemKind | Unset
        if isinstance(_kind,  Unset):
            kind = UNSET
        else:
            kind = MJMLThumbItemKind(_kind)




        _row = d.pop("row", UNSET)
        row: list[MJMLThumbItem] | Unset = UNSET
        if _row is not UNSET:
            row = []
            for row_item_data in _row:
                row_item = MJMLThumbItem.from_dict(row_item_data)



                row.append(row_item)


        w = d.pop("w", UNSET)

        mjml_thumb_item = cls(
            accent=accent,
            h=h,
            kind=kind,
            row=row,
            w=w,
        )

        return mjml_thumb_item

