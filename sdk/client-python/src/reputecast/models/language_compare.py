from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.language_compare_status import LanguageCompareStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.compare_row import CompareRow
  from ..models.compare_text import CompareText





T = TypeVar("T", bound="LanguageCompare")



@_attrs_define
class LanguageCompare:
    """ 
        Attributes:
            attention (int):
            blocks (int):
            language (str):
            preheader (CompareText):
            rows (list[CompareRow]):
            source (str):
            source_version (int):
            status (LanguageCompareStatus):
            subject (CompareText):
            translated (int):
            translated_by (str):
            translated_from (int):
            up_to_date (int):
            version (int):
            translated_at (datetime.datetime | Unset):
     """

    attention: int
    blocks: int
    language: str
    preheader: CompareText
    rows: list[CompareRow]
    source: str
    source_version: int
    status: LanguageCompareStatus
    subject: CompareText
    translated: int
    translated_by: str
    translated_from: int
    up_to_date: int
    version: int
    translated_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.compare_row import CompareRow # noqa: PLC0415
        from ..models.compare_text import CompareText # noqa: PLC0415
        attention = self.attention

        blocks = self.blocks

        language = self.language

        preheader = self.preheader.to_dict()

        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        source = self.source

        source_version = self.source_version

        status = self.status.value

        subject = self.subject.to_dict()

        translated = self.translated

        translated_by = self.translated_by

        translated_from = self.translated_from

        up_to_date = self.up_to_date

        version = self.version

        translated_at: str | Unset = UNSET
        if not isinstance(self.translated_at, Unset):
            translated_at = self.translated_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "attention": attention,
            "blocks": blocks,
            "language": language,
            "preheader": preheader,
            "rows": rows,
            "source": source,
            "sourceVersion": source_version,
            "status": status,
            "subject": subject,
            "translated": translated,
            "translatedBy": translated_by,
            "translatedFrom": translated_from,
            "upToDate": up_to_date,
            "version": version,
        })
        if translated_at is not UNSET:
            field_dict["translatedAt"] = translated_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.compare_row import CompareRow # noqa: PLC0415
        from ..models.compare_text import CompareText # noqa: PLC0415
        d = dict(src_dict)
        attention = d.pop("attention")

        blocks = d.pop("blocks")

        language = d.pop("language")

        preheader = CompareText.from_dict(d.pop("preheader"))




        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = CompareRow.from_dict(rows_item_data)



            rows.append(rows_item)


        source = d.pop("source")

        source_version = d.pop("sourceVersion")

        status = LanguageCompareStatus(d.pop("status"))




        subject = CompareText.from_dict(d.pop("subject"))




        translated = d.pop("translated")

        translated_by = d.pop("translatedBy")

        translated_from = d.pop("translatedFrom")

        up_to_date = d.pop("upToDate")

        version = d.pop("version")

        _translated_at = d.pop("translatedAt", UNSET)
        translated_at: datetime.datetime | Unset
        if isinstance(_translated_at,  Unset):
            translated_at = UNSET
        else:
            translated_at = datetime.datetime.fromisoformat(_translated_at)




        language_compare = cls(
            attention=attention,
            blocks=blocks,
            language=language,
            preheader=preheader,
            rows=rows,
            source=source,
            source_version=source_version,
            status=status,
            subject=subject,
            translated=translated,
            translated_by=translated_by,
            translated_from=translated_from,
            up_to_date=up_to_date,
            version=version,
            translated_at=translated_at,
        )

        return language_compare

