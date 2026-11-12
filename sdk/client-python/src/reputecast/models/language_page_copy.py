from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.language_page_copy_status import LanguagePageCopyStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="LanguagePageCopy")



@_attrs_define
class LanguagePageCopy:
    """ 
        Attributes:
            builtin_labels (bool):
            by (str):
            missing (int):
            status (LanguagePageCopyStatus):
            at (datetime.datetime | Unset):
     """

    builtin_labels: bool
    by: str
    missing: int
    status: LanguagePageCopyStatus
    at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        builtin_labels = self.builtin_labels

        by = self.by

        missing = self.missing

        status = self.status.value

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "builtinLabels": builtin_labels,
            "by": by,
            "missing": missing,
            "status": status,
        })
        if at is not UNSET:
            field_dict["at"] = at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        builtin_labels = d.pop("builtinLabels")

        by = d.pop("by")

        missing = d.pop("missing")

        status = LanguagePageCopyStatus(d.pop("status"))




        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        language_page_copy = cls(
            builtin_labels=builtin_labels,
            by=by,
            missing=missing,
            status=status,
            at=at,
        )

        return language_page_copy

