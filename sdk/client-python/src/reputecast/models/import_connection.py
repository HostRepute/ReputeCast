from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.import_source_list import ImportSourceList
  from ..models.import_template_counts import ImportTemplateCounts





T = TypeVar("T", bound="ImportConnection")



@_attrs_define
class ImportConnection:
    """ 
        Attributes:
            account (str):
            checked_at (datetime.datetime):
            lists (list[ImportSourceList]):
            base_url (str | Unset):
            data_center (str | Unset):
            templates (ImportTemplateCounts | Unset):
     """

    account: str
    checked_at: datetime.datetime
    lists: list[ImportSourceList]
    base_url: str | Unset = UNSET
    data_center: str | Unset = UNSET
    templates: ImportTemplateCounts | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_source_list import ImportSourceList # noqa: PLC0415
        from ..models.import_template_counts import ImportTemplateCounts # noqa: PLC0415
        account = self.account

        checked_at = self.checked_at.isoformat()

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        base_url = self.base_url

        data_center = self.data_center

        templates: dict[str, Any] | Unset = UNSET
        if not isinstance(self.templates, Unset):
            templates = self.templates.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "account": account,
            "checkedAt": checked_at,
            "lists": lists,
        })
        if base_url is not UNSET:
            field_dict["baseUrl"] = base_url
        if data_center is not UNSET:
            field_dict["dataCenter"] = data_center
        if templates is not UNSET:
            field_dict["templates"] = templates

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_source_list import ImportSourceList # noqa: PLC0415
        from ..models.import_template_counts import ImportTemplateCounts # noqa: PLC0415
        d = dict(src_dict)
        account = d.pop("account")

        checked_at = datetime.datetime.fromisoformat(d.pop("checkedAt"))




        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = ImportSourceList.from_dict(lists_item_data)



            lists.append(lists_item)


        base_url = d.pop("baseUrl", UNSET)

        data_center = d.pop("dataCenter", UNSET)

        _templates = d.pop("templates", UNSET)
        templates: ImportTemplateCounts | Unset
        if isinstance(_templates,  Unset):
            templates = UNSET
        else:
            templates = ImportTemplateCounts.from_dict(_templates)




        import_connection = cls(
            account=account,
            checked_at=checked_at,
            lists=lists,
            base_url=base_url,
            data_center=data_center,
            templates=templates,
        )

        return import_connection

