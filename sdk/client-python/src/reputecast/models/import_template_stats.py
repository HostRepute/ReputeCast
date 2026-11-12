from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ImportTemplateStats")



@_attrs_define
class ImportTemplateStats:
    """ 
        Attributes:
            manual (int):
            saved (int): Marketing and other templates added to the saved templates
            skipped (int):
            transactional (int): Transactional templates imported as drafts
     """

    manual: int
    saved: int
    skipped: int
    transactional: int





    def to_dict(self) -> dict[str, Any]:
        manual = self.manual

        saved = self.saved

        skipped = self.skipped

        transactional = self.transactional


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "manual": manual,
            "saved": saved,
            "skipped": skipped,
            "transactional": transactional,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        manual = d.pop("manual")

        saved = d.pop("saved")

        skipped = d.pop("skipped")

        transactional = d.pop("transactional")

        import_template_stats = cls(
            manual=manual,
            saved=saved,
            skipped=skipped,
            transactional=transactional,
        )

        return import_template_stats

