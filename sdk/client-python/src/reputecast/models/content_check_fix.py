from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.content_check_fix_kind import ContentCheckFixKind
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ContentCheckFix")



@_attrs_define
class ContentCheckFix:
    """ 
        Attributes:
            kind (ContentCheckFixKind):
            label (str):
            path (list[int] | Unset):
            value (str | Unset):
     """

    kind: ContentCheckFixKind
    label: str
    path: list[int] | Unset = UNSET
    value: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        label = self.label

        path: list[int] | Unset = UNSET
        if not isinstance(self.path, Unset):
            path = self.path



        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "label": label,
        })
        if path is not UNSET:
            field_dict["path"] = path
        if value is not UNSET:
            field_dict["value"] = value

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = ContentCheckFixKind(d.pop("kind"))




        label = d.pop("label")

        path = cast(list[int], d.pop("path", UNSET))


        value = d.pop("value", UNSET)

        content_check_fix = cls(
            kind=kind,
            label=label,
            path=path,
            value=value,
        )

        return content_check_fix

