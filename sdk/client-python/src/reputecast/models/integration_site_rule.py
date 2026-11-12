from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.integration_list_option import IntegrationListOption
  from ..models.integration_option import IntegrationOption
  from ..models.word_press_category import WordPressCategory





T = TypeVar("T", bound="IntegrationSiteRule")



@_attrs_define
class IntegrationSiteRule:
    """ 
        Attributes:
            categories (list[WordPressCategory]):
            enabled (bool):
            list_ids (list[str]):
            set_ (bool): A rule was set up (even if it's off now)
            list_ (IntegrationListOption | Unset):
            notify (IntegrationOption | Unset):
            sender (IntegrationOption | Unset):
            template (IntegrationOption | Unset):
     """

    categories: list[WordPressCategory]
    enabled: bool
    list_ids: list[str]
    set_: bool
    list_: IntegrationListOption | Unset = UNSET
    notify: IntegrationOption | Unset = UNSET
    sender: IntegrationOption | Unset = UNSET
    template: IntegrationOption | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_list_option import IntegrationListOption # noqa: PLC0415
        from ..models.integration_option import IntegrationOption # noqa: PLC0415
        from ..models.word_press_category import WordPressCategory # noqa: PLC0415
        categories = []
        for categories_item_data in self.categories:
            categories_item = categories_item_data.to_dict()
            categories.append(categories_item)



        enabled = self.enabled

        list_ids = self.list_ids



        set_ = self.set_

        list_: dict[str, Any] | Unset = UNSET
        if not isinstance(self.list_, Unset):
            list_ = self.list_.to_dict()

        notify: dict[str, Any] | Unset = UNSET
        if not isinstance(self.notify, Unset):
            notify = self.notify.to_dict()

        sender: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sender, Unset):
            sender = self.sender.to_dict()

        template: dict[str, Any] | Unset = UNSET
        if not isinstance(self.template, Unset):
            template = self.template.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "categories": categories,
            "enabled": enabled,
            "listIds": list_ids,
            "set": set_,
        })
        if list_ is not UNSET:
            field_dict["list"] = list_
        if notify is not UNSET:
            field_dict["notify"] = notify
        if sender is not UNSET:
            field_dict["sender"] = sender
        if template is not UNSET:
            field_dict["template"] = template

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_list_option import IntegrationListOption # noqa: PLC0415
        from ..models.integration_option import IntegrationOption # noqa: PLC0415
        from ..models.word_press_category import WordPressCategory # noqa: PLC0415
        d = dict(src_dict)
        categories = []
        _categories = d.pop("categories")
        for categories_item_data in (_categories):
            categories_item = WordPressCategory.from_dict(categories_item_data)



            categories.append(categories_item)


        enabled = d.pop("enabled")

        list_ids = cast(list[str], d.pop("listIds"))


        set_ = d.pop("set")

        _list_ = d.pop("list", UNSET)
        list_: IntegrationListOption | Unset
        if isinstance(_list_,  Unset):
            list_ = UNSET
        else:
            list_ = IntegrationListOption.from_dict(_list_)




        _notify = d.pop("notify", UNSET)
        notify: IntegrationOption | Unset
        if isinstance(_notify,  Unset):
            notify = UNSET
        else:
            notify = IntegrationOption.from_dict(_notify)




        _sender = d.pop("sender", UNSET)
        sender: IntegrationOption | Unset
        if isinstance(_sender,  Unset):
            sender = UNSET
        else:
            sender = IntegrationOption.from_dict(_sender)




        _template = d.pop("template", UNSET)
        template: IntegrationOption | Unset
        if isinstance(_template,  Unset):
            template = UNSET
        else:
            template = IntegrationOption.from_dict(_template)




        integration_site_rule = cls(
            categories=categories,
            enabled=enabled,
            list_ids=list_ids,
            set_=set_,
            list_=list_,
            notify=notify,
            sender=sender,
            template=template,
        )

        return integration_site_rule

