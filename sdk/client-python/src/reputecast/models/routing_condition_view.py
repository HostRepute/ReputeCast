from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.routing_condition_view_field import RoutingConditionViewField
from ..types import UNSET, Unset
from typing import cast
from uuid import UUID

if TYPE_CHECKING:
  from ..models.routing_split import RoutingSplit





T = TypeVar("T", bound="RoutingConditionView")



@_attrs_define
class RoutingConditionView:
    """ 
        Attributes:
            field (RoutingConditionViewField): category: a transactional category (Transactional stream only)
            value_labels (list[str]):
            values (list[str]):
            note (str | Unset):
            provider_id (UUID | Unset):
            splits (list[RoutingSplit] | Unset):
     """

    field: RoutingConditionViewField
    value_labels: list[str]
    values: list[str]
    note: str | Unset = UNSET
    provider_id: UUID | Unset = UNSET
    splits: list[RoutingSplit] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.routing_split import RoutingSplit # noqa: PLC0415
        field = self.field.value

        value_labels = self.value_labels



        values = self.values



        note = self.note

        provider_id: str | Unset = UNSET
        if not isinstance(self.provider_id, Unset):
            provider_id = str(self.provider_id)

        splits: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.splits, Unset):
            splits = []
            for splits_item_data in self.splits:
                splits_item = splits_item_data.to_dict()
                splits.append(splits_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "field": field,
            "valueLabels": value_labels,
            "values": values,
        })
        if note is not UNSET:
            field_dict["note"] = note
        if provider_id is not UNSET:
            field_dict["providerId"] = provider_id
        if splits is not UNSET:
            field_dict["splits"] = splits

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.routing_split import RoutingSplit # noqa: PLC0415
        d = dict(src_dict)
        field = RoutingConditionViewField(d.pop("field"))




        value_labels = cast(list[str], d.pop("valueLabels"))


        values = cast(list[str], d.pop("values"))


        note = d.pop("note", UNSET)

        _provider_id = d.pop("providerId", UNSET)
        provider_id: UUID | Unset
        if isinstance(_provider_id,  Unset):
            provider_id = UNSET
        else:
            provider_id = UUID(_provider_id)




        _splits = d.pop("splits", UNSET)
        splits: list[RoutingSplit] | Unset = UNSET
        if _splits is not UNSET:
            splits = []
            for splits_item_data in _splits:
                splits_item = RoutingSplit.from_dict(splits_item_data)



                splits.append(splits_item)


        routing_condition_view = cls(
            field=field,
            value_labels=value_labels,
            values=values,
            note=note,
            provider_id=provider_id,
            splits=splits,
        )

        return routing_condition_view

