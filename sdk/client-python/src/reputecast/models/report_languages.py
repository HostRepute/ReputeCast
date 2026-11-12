from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.report_language_count import ReportLanguageCount
  from ..models.report_language_fallback import ReportLanguageFallback
  from ..models.report_language_option import ReportLanguageOption
  from ..models.report_language_row import ReportLanguageRow





T = TypeVar("T", bound="ReportLanguages")



@_attrs_define
class ReportLanguages:
    """ 
        Attributes:
            click_maps (list[ReportLanguageOption]):
            languages (int):
            rows (list[ReportLanguageRow]):
            sent (list[ReportLanguageCount]):
            versions (int):
            fallback (ReportLanguageFallback | Unset):
     """

    click_maps: list[ReportLanguageOption]
    languages: int
    rows: list[ReportLanguageRow]
    sent: list[ReportLanguageCount]
    versions: int
    fallback: ReportLanguageFallback | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_language_count import ReportLanguageCount # noqa: PLC0415
        from ..models.report_language_fallback import ReportLanguageFallback # noqa: PLC0415
        from ..models.report_language_option import ReportLanguageOption # noqa: PLC0415
        from ..models.report_language_row import ReportLanguageRow # noqa: PLC0415
        click_maps = []
        for click_maps_item_data in self.click_maps:
            click_maps_item = click_maps_item_data.to_dict()
            click_maps.append(click_maps_item)



        languages = self.languages

        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        sent = []
        for sent_item_data in self.sent:
            sent_item = sent_item_data.to_dict()
            sent.append(sent_item)



        versions = self.versions

        fallback: dict[str, Any] | Unset = UNSET
        if not isinstance(self.fallback, Unset):
            fallback = self.fallback.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clickMaps": click_maps,
            "languages": languages,
            "rows": rows,
            "sent": sent,
            "versions": versions,
        })
        if fallback is not UNSET:
            field_dict["fallback"] = fallback

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_language_count import ReportLanguageCount # noqa: PLC0415
        from ..models.report_language_fallback import ReportLanguageFallback # noqa: PLC0415
        from ..models.report_language_option import ReportLanguageOption # noqa: PLC0415
        from ..models.report_language_row import ReportLanguageRow # noqa: PLC0415
        d = dict(src_dict)
        click_maps = []
        _click_maps = d.pop("clickMaps")
        for click_maps_item_data in (_click_maps):
            click_maps_item = ReportLanguageOption.from_dict(click_maps_item_data)



            click_maps.append(click_maps_item)


        languages = d.pop("languages")

        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = ReportLanguageRow.from_dict(rows_item_data)



            rows.append(rows_item)


        sent = []
        _sent = d.pop("sent")
        for sent_item_data in (_sent):
            sent_item = ReportLanguageCount.from_dict(sent_item_data)



            sent.append(sent_item)


        versions = d.pop("versions")

        _fallback = d.pop("fallback", UNSET)
        fallback: ReportLanguageFallback | Unset
        if isinstance(_fallback,  Unset):
            fallback = UNSET
        else:
            fallback = ReportLanguageFallback.from_dict(_fallback)




        report_languages = cls(
            click_maps=click_maps,
            languages=languages,
            rows=rows,
            sent=sent,
            versions=versions,
            fallback=fallback,
        )

        return report_languages

