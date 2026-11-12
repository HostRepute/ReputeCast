from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.report_link import ReportLink
  from ..models.report_link_group import ReportLinkGroup
  from ..models.report_link_row import ReportLinkRow
  from ..models.report_variant_click_map import ReportVariantClickMap





T = TypeVar("T", bound="ReportClickMap")



@_attrs_define
class ReportClickMap:
    """ 
        Attributes:
            available (bool):
            bot_total (int): Clicks the bot filter left out
            html (str):
            links (list[ReportLink]):
            per_variant (bool):
            rows (list[ReportLinkRow]):
            threshold (float): Links under this share (percent) are grouped as Other
            total (int): Human clicks on the links, each person once per link
            variants (list[ReportVariantClickMap]):
            language (str | Unset):
            other (ReportLinkGroup | Unset):
     """

    available: bool
    bot_total: int
    html: str
    links: list[ReportLink]
    per_variant: bool
    rows: list[ReportLinkRow]
    threshold: float
    total: int
    variants: list[ReportVariantClickMap]
    language: str | Unset = UNSET
    other: ReportLinkGroup | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_link import ReportLink # noqa: PLC0415
        from ..models.report_link_group import ReportLinkGroup # noqa: PLC0415
        from ..models.report_link_row import ReportLinkRow # noqa: PLC0415
        from ..models.report_variant_click_map import ReportVariantClickMap # noqa: PLC0415
        available = self.available

        bot_total = self.bot_total

        html = self.html

        links = []
        for links_item_data in self.links:
            links_item = links_item_data.to_dict()
            links.append(links_item)



        per_variant = self.per_variant

        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        threshold = self.threshold

        total = self.total

        variants = []
        for variants_item_data in self.variants:
            variants_item = variants_item_data.to_dict()
            variants.append(variants_item)



        language = self.language

        other: dict[str, Any] | Unset = UNSET
        if not isinstance(self.other, Unset):
            other = self.other.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
            "botTotal": bot_total,
            "html": html,
            "links": links,
            "perVariant": per_variant,
            "rows": rows,
            "threshold": threshold,
            "total": total,
            "variants": variants,
        })
        if language is not UNSET:
            field_dict["language"] = language
        if other is not UNSET:
            field_dict["other"] = other

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_link import ReportLink # noqa: PLC0415
        from ..models.report_link_group import ReportLinkGroup # noqa: PLC0415
        from ..models.report_link_row import ReportLinkRow # noqa: PLC0415
        from ..models.report_variant_click_map import ReportVariantClickMap # noqa: PLC0415
        d = dict(src_dict)
        available = d.pop("available")

        bot_total = d.pop("botTotal")

        html = d.pop("html")

        links = []
        _links = d.pop("links")
        for links_item_data in (_links):
            links_item = ReportLink.from_dict(links_item_data)



            links.append(links_item)


        per_variant = d.pop("perVariant")

        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = ReportLinkRow.from_dict(rows_item_data)



            rows.append(rows_item)


        threshold = d.pop("threshold")

        total = d.pop("total")

        variants = []
        _variants = d.pop("variants")
        for variants_item_data in (_variants):
            variants_item = ReportVariantClickMap.from_dict(variants_item_data)



            variants.append(variants_item)


        language = d.pop("language", UNSET)

        _other = d.pop("other", UNSET)
        other: ReportLinkGroup | Unset
        if isinstance(_other,  Unset):
            other = UNSET
        else:
            other = ReportLinkGroup.from_dict(_other)




        report_click_map = cls(
            available=available,
            bot_total=bot_total,
            html=html,
            links=links,
            per_variant=per_variant,
            rows=rows,
            threshold=threshold,
            total=total,
            variants=variants,
            language=language,
            other=other,
        )

        return report_click_map

