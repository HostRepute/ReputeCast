from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.report_comparisons_unavailable import ReportComparisonsUnavailable
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.report_comparison import ReportComparison





T = TypeVar("T", bound="ReportComparisons")



@_attrs_define
class ReportComparisons:
    """ 
        Attributes:
            issues (int):
            tiles (list[ReportComparison]):
            unavailable (ReportComparisonsUnavailable | Unset):
     """

    issues: int
    tiles: list[ReportComparison]
    unavailable: ReportComparisonsUnavailable | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_comparison import ReportComparison # noqa: PLC0415
        issues = self.issues

        tiles = []
        for tiles_item_data in self.tiles:
            tiles_item = tiles_item_data.to_dict()
            tiles.append(tiles_item)



        unavailable: str | Unset = UNSET
        if not isinstance(self.unavailable, Unset):
            unavailable = self.unavailable.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "issues": issues,
            "tiles": tiles,
        })
        if unavailable is not UNSET:
            field_dict["unavailable"] = unavailable

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_comparison import ReportComparison # noqa: PLC0415
        d = dict(src_dict)
        issues = d.pop("issues")

        tiles = []
        _tiles = d.pop("tiles")
        for tiles_item_data in (_tiles):
            tiles_item = ReportComparison.from_dict(tiles_item_data)



            tiles.append(tiles_item)


        _unavailable = d.pop("unavailable", UNSET)
        unavailable: ReportComparisonsUnavailable | Unset
        if isinstance(_unavailable,  Unset):
            unavailable = UNSET
        else:
            unavailable = ReportComparisonsUnavailable(_unavailable)




        report_comparisons = cls(
            issues=issues,
            tiles=tiles,
            unavailable=unavailable,
        )

        return report_comparisons

