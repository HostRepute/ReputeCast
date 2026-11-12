from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.connect_import_source import ConnectImportSource
  from ..models.connect_suppressions import ConnectSuppressions





T = TypeVar("T", bound="ConnectContacts")



@_attrs_define
class ConnectContacts:
    """ 
        Attributes:
            lists (list[str]):
            pull (bool):
            sources (list[ConnectImportSource]):
            suggested (list[str]):
            suppressions (ConnectSuppressions):
     """

    lists: list[str]
    pull: bool
    sources: list[ConnectImportSource]
    suggested: list[str]
    suppressions: ConnectSuppressions





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_import_source import ConnectImportSource # noqa: PLC0415
        from ..models.connect_suppressions import ConnectSuppressions # noqa: PLC0415
        lists = self.lists



        pull = self.pull

        sources = []
        for sources_item_data in self.sources:
            sources_item = sources_item_data.to_dict()
            sources.append(sources_item)



        suggested = self.suggested



        suppressions = self.suppressions.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "lists": lists,
            "pull": pull,
            "sources": sources,
            "suggested": suggested,
            "suppressions": suppressions,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_import_source import ConnectImportSource # noqa: PLC0415
        from ..models.connect_suppressions import ConnectSuppressions # noqa: PLC0415
        d = dict(src_dict)
        lists = cast(list[str], d.pop("lists"))


        pull = d.pop("pull")

        sources = []
        _sources = d.pop("sources")
        for sources_item_data in (_sources):
            sources_item = ConnectImportSource.from_dict(sources_item_data)



            sources.append(sources_item)


        suggested = cast(list[str], d.pop("suggested"))


        suppressions = ConnectSuppressions.from_dict(d.pop("suppressions"))




        connect_contacts = cls(
            lists=lists,
            pull=pull,
            sources=sources,
            suggested=suggested,
            suppressions=suppressions,
        )

        return connect_contacts

