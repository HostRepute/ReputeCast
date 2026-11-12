from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_entry_key import IntegrationEntryKey
from ..models.integration_entry_status import IntegrationEntryStatus
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.integration_facts import IntegrationFacts
  from ..models.integration_reason import IntegrationReason





T = TypeVar("T", bound="IntegrationEntry")



@_attrs_define
class IntegrationEntry:
    """ 
        Attributes:
            facts (IntegrationFacts):
            key (IntegrationEntryKey):
            reasons (list[IntegrationReason]):
            status (IntegrationEntryStatus):
            update_available (bool):
            connection_id (str | Unset):
     """

    facts: IntegrationFacts
    key: IntegrationEntryKey
    reasons: list[IntegrationReason]
    status: IntegrationEntryStatus
    update_available: bool
    connection_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_facts import IntegrationFacts # noqa: PLC0415
        from ..models.integration_reason import IntegrationReason # noqa: PLC0415
        facts = self.facts.to_dict()

        key = self.key.value

        reasons = []
        for reasons_item_data in self.reasons:
            reasons_item = reasons_item_data.to_dict()
            reasons.append(reasons_item)



        status = self.status.value

        update_available = self.update_available

        connection_id = self.connection_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "facts": facts,
            "key": key,
            "reasons": reasons,
            "status": status,
            "updateAvailable": update_available,
        })
        if connection_id is not UNSET:
            field_dict["connectionId"] = connection_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_facts import IntegrationFacts # noqa: PLC0415
        from ..models.integration_reason import IntegrationReason # noqa: PLC0415
        d = dict(src_dict)
        facts = IntegrationFacts.from_dict(d.pop("facts"))




        key = IntegrationEntryKey(d.pop("key"))




        reasons = []
        _reasons = d.pop("reasons")
        for reasons_item_data in (_reasons):
            reasons_item = IntegrationReason.from_dict(reasons_item_data)



            reasons.append(reasons_item)


        status = IntegrationEntryStatus(d.pop("status"))




        update_available = d.pop("updateAvailable")

        connection_id = d.pop("connectionId", UNSET)

        integration_entry = cls(
            facts=facts,
            key=key,
            reasons=reasons,
            status=status,
            update_available=update_available,
            connection_id=connection_id,
        )

        return integration_entry

