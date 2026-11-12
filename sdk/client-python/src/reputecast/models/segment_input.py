from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.segment_input_mode import SegmentInputMode
from ..models.segment_input_refresh import SegmentInputRefresh
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.segment_definition import SegmentDefinition





T = TypeVar("T", bound="SegmentInput")



@_attrs_define
class SegmentInput:
    """ 
        Attributes:
            mode (SegmentInputMode):
            definition (SegmentDefinition | Unset):
            description (str | Unset):
            name (str | Unset):
            refresh (SegmentInputRefresh | Unset):
            sql (str | Unset):
     """

    mode: SegmentInputMode
    definition: SegmentDefinition | Unset = UNSET
    description: str | Unset = UNSET
    name: str | Unset = UNSET
    refresh: SegmentInputRefresh | Unset = UNSET
    sql: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.segment_definition import SegmentDefinition # noqa: PLC0415
        mode = self.mode.value

        definition: dict[str, Any] | Unset = UNSET
        if not isinstance(self.definition, Unset):
            definition = self.definition.to_dict()

        description = self.description

        name = self.name

        refresh: str | Unset = UNSET
        if not isinstance(self.refresh, Unset):
            refresh = self.refresh.value


        sql = self.sql


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
        })
        if definition is not UNSET:
            field_dict["definition"] = definition
        if description is not UNSET:
            field_dict["description"] = description
        if name is not UNSET:
            field_dict["name"] = name
        if refresh is not UNSET:
            field_dict["refresh"] = refresh
        if sql is not UNSET:
            field_dict["sql"] = sql

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.segment_definition import SegmentDefinition # noqa: PLC0415
        d = dict(src_dict)
        mode = SegmentInputMode(d.pop("mode"))




        _definition = d.pop("definition", UNSET)
        definition: SegmentDefinition | Unset
        if isinstance(_definition,  Unset):
            definition = UNSET
        else:
            definition = SegmentDefinition.from_dict(_definition)




        description = d.pop("description", UNSET)

        name = d.pop("name", UNSET)

        _refresh = d.pop("refresh", UNSET)
        refresh: SegmentInputRefresh | Unset
        if isinstance(_refresh,  Unset):
            refresh = UNSET
        else:
            refresh = SegmentInputRefresh(_refresh)




        sql = d.pop("sql", UNSET)

        segment_input = cls(
            mode=mode,
            definition=definition,
            description=description,
            name=name,
            refresh=refresh,
            sql=sql,
        )

        return segment_input

