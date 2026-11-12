from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.growth_import_item import GrowthImportItem





T = TypeVar("T", bound="GrowthTiles")



@_attrs_define
class GrowthTiles:
    """ 
        Attributes:
            import_items (list[GrowthImportItem]):
            imports (int):
            joined (int):
            left (int):
            net (int):
            net_pct (float):
     """

    import_items: list[GrowthImportItem]
    imports: int
    joined: int
    left: int
    net: int
    net_pct: float





    def to_dict(self) -> dict[str, Any]:
        from ..models.growth_import_item import GrowthImportItem # noqa: PLC0415
        import_items = []
        for import_items_item_data in self.import_items:
            import_items_item = import_items_item_data.to_dict()
            import_items.append(import_items_item)



        imports = self.imports

        joined = self.joined

        left = self.left

        net = self.net

        net_pct = self.net_pct


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "importItems": import_items,
            "imports": imports,
            "joined": joined,
            "left": left,
            "net": net,
            "netPct": net_pct,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.growth_import_item import GrowthImportItem # noqa: PLC0415
        d = dict(src_dict)
        import_items = []
        _import_items = d.pop("importItems")
        for import_items_item_data in (_import_items):
            import_items_item = GrowthImportItem.from_dict(import_items_item_data)



            import_items.append(import_items_item)


        imports = d.pop("imports")

        joined = d.pop("joined")

        left = d.pop("left")

        net = d.pop("net")

        net_pct = d.pop("netPct")

        growth_tiles = cls(
            import_items=import_items,
            imports=imports,
            joined=joined,
            left=left,
            net=net,
            net_pct=net_pct,
        )

        return growth_tiles

