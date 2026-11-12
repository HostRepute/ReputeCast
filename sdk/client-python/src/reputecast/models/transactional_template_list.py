from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_template_list_period import TransactionalTemplateListPeriod
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_template_list_item import TransactionalTemplateListItem





T = TypeVar("T", bound="TransactionalTemplateList")



@_attrs_define
class TransactionalTemplateList:
    """ 
        Attributes:
            can_edit (bool):
            can_publish (bool):
            items (list[TransactionalTemplateListItem]):
            period (TransactionalTemplateListPeriod):
            read_only_reason (str | Unset):
     """

    can_edit: bool
    can_publish: bool
    items: list[TransactionalTemplateListItem]
    period: TransactionalTemplateListPeriod
    read_only_reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_template_list_item import TransactionalTemplateListItem # noqa: PLC0415
        can_edit = self.can_edit

        can_publish = self.can_publish

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        period = self.period.value

        read_only_reason = self.read_only_reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "canPublish": can_publish,
            "items": items,
            "period": period,
        })
        if read_only_reason is not UNSET:
            field_dict["readOnlyReason"] = read_only_reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_template_list_item import TransactionalTemplateListItem # noqa: PLC0415
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        can_publish = d.pop("canPublish")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = TransactionalTemplateListItem.from_dict(items_item_data)



            items.append(items_item)


        period = TransactionalTemplateListPeriod(d.pop("period"))




        read_only_reason = d.pop("readOnlyReason", UNSET)

        transactional_template_list = cls(
            can_edit=can_edit,
            can_publish=can_publish,
            items=items,
            period=period,
            read_only_reason=read_only_reason,
        )

        return transactional_template_list

