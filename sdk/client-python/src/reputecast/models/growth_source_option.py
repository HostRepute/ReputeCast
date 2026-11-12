from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.growth_source import GrowthSource





T = TypeVar("T", bound="GrowthSourceOption")



@_attrs_define
class GrowthSourceOption:
    """ 
        Attributes:
            contacts (int):
            source (GrowthSource):
     """

    contacts: int
    source: GrowthSource





    def to_dict(self) -> dict[str, Any]:
        from ..models.growth_source import GrowthSource # noqa: PLC0415
        contacts = self.contacts

        source = self.source.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contacts": contacts,
            "source": source,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.growth_source import GrowthSource # noqa: PLC0415
        d = dict(src_dict)
        contacts = d.pop("contacts")

        source = GrowthSource.from_dict(d.pop("source"))




        growth_source_option = cls(
            contacts=contacts,
            source=source,
        )

        return growth_source_option

