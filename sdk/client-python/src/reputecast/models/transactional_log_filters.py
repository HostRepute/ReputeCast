from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_ref import TransactionalRef





T = TypeVar("T", bound="TransactionalLogFilters")



@_attrs_define
class TransactionalLogFilters:
    """ 
        Attributes:
            keys (list[TransactionalRef]):
            providers (list[TransactionalRef]):
            templates (list[TransactionalRef]): id is the template's ID; raw is mail that arrived by SMTP
     """

    keys: list[TransactionalRef]
    providers: list[TransactionalRef]
    templates: list[TransactionalRef]





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_ref import TransactionalRef # noqa: PLC0415
        keys = []
        for keys_item_data in self.keys:
            keys_item = keys_item_data.to_dict()
            keys.append(keys_item)



        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        templates = []
        for templates_item_data in self.templates:
            templates_item = templates_item_data.to_dict()
            templates.append(templates_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "keys": keys,
            "providers": providers,
            "templates": templates,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_ref import TransactionalRef # noqa: PLC0415
        d = dict(src_dict)
        keys = []
        _keys = d.pop("keys")
        for keys_item_data in (_keys):
            keys_item = TransactionalRef.from_dict(keys_item_data)



            keys.append(keys_item)


        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = TransactionalRef.from_dict(providers_item_data)



            providers.append(providers_item)


        templates = []
        _templates = d.pop("templates")
        for templates_item_data in (_templates):
            templates_item = TransactionalRef.from_dict(templates_item_data)



            templates.append(templates_item)


        transactional_log_filters = cls(
            keys=keys,
            providers=providers,
            templates=templates,
        )

        return transactional_log_filters

