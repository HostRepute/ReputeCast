from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_create_body_template import AutomationCreateBodyTemplate
from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="AutomationCreateBody")



@_attrs_define
class AutomationCreateBody:
    """ 
        Attributes:
            list_ (UUID | Unset): For a template that starts on a list (welcome-series): the list; otherwise the workspace's
                first list
            name (str | Unset):
            store (UUID | Unset): For a template that starts on a store's events (abandoned-cart): the connected store;
                otherwise the workspace's first
            template (AutomationCreateBodyTemplate | Unset): Starter template; omit for a blank canvas
     """

    list_: UUID | Unset = UNSET
    name: str | Unset = UNSET
    store: UUID | Unset = UNSET
    template: AutomationCreateBodyTemplate | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        list_: str | Unset = UNSET
        if not isinstance(self.list_, Unset):
            list_ = str(self.list_)

        name = self.name

        store: str | Unset = UNSET
        if not isinstance(self.store, Unset):
            store = str(self.store)

        template: str | Unset = UNSET
        if not isinstance(self.template, Unset):
            template = self.template.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if list_ is not UNSET:
            field_dict["list"] = list_
        if name is not UNSET:
            field_dict["name"] = name
        if store is not UNSET:
            field_dict["store"] = store
        if template is not UNSET:
            field_dict["template"] = template

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _list_ = d.pop("list", UNSET)
        list_: UUID | Unset
        if isinstance(_list_,  Unset):
            list_ = UNSET
        else:
            list_ = UUID(_list_)




        name = d.pop("name", UNSET)

        _store = d.pop("store", UNSET)
        store: UUID | Unset
        if isinstance(_store,  Unset):
            store = UNSET
        else:
            store = UUID(_store)




        _template = d.pop("template", UNSET)
        template: AutomationCreateBodyTemplate | Unset
        if isinstance(_template,  Unset):
            template = UNSET
        else:
            template = AutomationCreateBodyTemplate(_template)




        automation_create_body = cls(
            list_=list_,
            name=name,
            store=store,
            template=template,
        )

        return automation_create_body

