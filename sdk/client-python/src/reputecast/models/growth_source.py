from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="GrowthSource")



@_attrs_define
class GrowthSource:
    """ 
        Attributes:
            hosts (list[str]):
            imported (bool):
            key (str):
            kind (str):
            name (str):
            carried (str | Unset):
            carried_label (str | Unset):
            first_join (datetime.datetime | Unset):
            form_kind (str | Unset):
            import_count (int | Unset):
            imported_at (datetime.datetime | Unset):
            integration (str | Unset):
            key_prefix (str | Unset):
            list_choice (bool | Unset):
            took_import_day (int | Unset):
            url (str | Unset):
            workflow (str | Unset):
     """

    hosts: list[str]
    imported: bool
    key: str
    kind: str
    name: str
    carried: str | Unset = UNSET
    carried_label: str | Unset = UNSET
    first_join: datetime.datetime | Unset = UNSET
    form_kind: str | Unset = UNSET
    import_count: int | Unset = UNSET
    imported_at: datetime.datetime | Unset = UNSET
    integration: str | Unset = UNSET
    key_prefix: str | Unset = UNSET
    list_choice: bool | Unset = UNSET
    took_import_day: int | Unset = UNSET
    url: str | Unset = UNSET
    workflow: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        hosts = self.hosts



        imported = self.imported

        key = self.key

        kind = self.kind

        name = self.name

        carried = self.carried

        carried_label = self.carried_label

        first_join: str | Unset = UNSET
        if not isinstance(self.first_join, Unset):
            first_join = self.first_join.isoformat()

        form_kind = self.form_kind

        import_count = self.import_count

        imported_at: str | Unset = UNSET
        if not isinstance(self.imported_at, Unset):
            imported_at = self.imported_at.isoformat()

        integration = self.integration

        key_prefix = self.key_prefix

        list_choice = self.list_choice

        took_import_day = self.took_import_day

        url = self.url

        workflow = self.workflow


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hosts": hosts,
            "imported": imported,
            "key": key,
            "kind": kind,
            "name": name,
        })
        if carried is not UNSET:
            field_dict["carried"] = carried
        if carried_label is not UNSET:
            field_dict["carriedLabel"] = carried_label
        if first_join is not UNSET:
            field_dict["firstJoin"] = first_join
        if form_kind is not UNSET:
            field_dict["formKind"] = form_kind
        if import_count is not UNSET:
            field_dict["importCount"] = import_count
        if imported_at is not UNSET:
            field_dict["importedAt"] = imported_at
        if integration is not UNSET:
            field_dict["integration"] = integration
        if key_prefix is not UNSET:
            field_dict["keyPrefix"] = key_prefix
        if list_choice is not UNSET:
            field_dict["listChoice"] = list_choice
        if took_import_day is not UNSET:
            field_dict["tookImportDay"] = took_import_day
        if url is not UNSET:
            field_dict["url"] = url
        if workflow is not UNSET:
            field_dict["workflow"] = workflow

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        hosts = cast(list[str], d.pop("hosts"))


        imported = d.pop("imported")

        key = d.pop("key")

        kind = d.pop("kind")

        name = d.pop("name")

        carried = d.pop("carried", UNSET)

        carried_label = d.pop("carriedLabel", UNSET)

        _first_join = d.pop("firstJoin", UNSET)
        first_join: datetime.datetime | Unset
        if isinstance(_first_join,  Unset):
            first_join = UNSET
        else:
            first_join = datetime.datetime.fromisoformat(_first_join)




        form_kind = d.pop("formKind", UNSET)

        import_count = d.pop("importCount", UNSET)

        _imported_at = d.pop("importedAt", UNSET)
        imported_at: datetime.datetime | Unset
        if isinstance(_imported_at,  Unset):
            imported_at = UNSET
        else:
            imported_at = datetime.datetime.fromisoformat(_imported_at)




        integration = d.pop("integration", UNSET)

        key_prefix = d.pop("keyPrefix", UNSET)

        list_choice = d.pop("listChoice", UNSET)

        took_import_day = d.pop("tookImportDay", UNSET)

        url = d.pop("url", UNSET)

        workflow = d.pop("workflow", UNSET)

        growth_source = cls(
            hosts=hosts,
            imported=imported,
            key=key,
            kind=kind,
            name=name,
            carried=carried,
            carried_label=carried_label,
            first_join=first_join,
            form_kind=form_kind,
            import_count=import_count,
            imported_at=imported_at,
            integration=integration,
            key_prefix=key_prefix,
            list_choice=list_choice,
            took_import_day=took_import_day,
            url=url,
            workflow=workflow,
        )

        return growth_source

