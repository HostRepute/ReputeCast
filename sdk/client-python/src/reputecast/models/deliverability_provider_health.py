from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_provider_health_tone import DeliverabilityProviderHealthTone
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.deliverability_provider_note import DeliverabilityProviderNote
  from ..models.deliverability_rate_limit import DeliverabilityRateLimit





T = TypeVar("T", bound="DeliverabilityProviderHealth")



@_attrs_define
class DeliverabilityProviderHealth:
    """ 
        Attributes:
            adapter (str):
            badge (str): Healthy · no enforcement
            bounce (DeliverabilityRateLimit):
            complaint (DeliverabilityRateLimit):
            name (str): Postal (self-hosted)
            provider_id (str):
            status (str):
            tone (DeliverabilityProviderHealthTone):
            not_judged (str | Unset):
            note (DeliverabilityProviderNote | Unset):
     """

    adapter: str
    badge: str
    bounce: DeliverabilityRateLimit
    complaint: DeliverabilityRateLimit
    name: str
    provider_id: str
    status: str
    tone: DeliverabilityProviderHealthTone
    not_judged: str | Unset = UNSET
    note: DeliverabilityProviderNote | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.deliverability_provider_note import DeliverabilityProviderNote # noqa: PLC0415
        from ..models.deliverability_rate_limit import DeliverabilityRateLimit # noqa: PLC0415
        adapter = self.adapter

        badge = self.badge

        bounce = self.bounce.to_dict()

        complaint = self.complaint.to_dict()

        name = self.name

        provider_id = self.provider_id

        status = self.status

        tone = self.tone.value

        not_judged = self.not_judged

        note: dict[str, Any] | Unset = UNSET
        if not isinstance(self.note, Unset):
            note = self.note.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "badge": badge,
            "bounce": bounce,
            "complaint": complaint,
            "name": name,
            "providerId": provider_id,
            "status": status,
            "tone": tone,
        })
        if not_judged is not UNSET:
            field_dict["notJudged"] = not_judged
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.deliverability_provider_note import DeliverabilityProviderNote # noqa: PLC0415
        from ..models.deliverability_rate_limit import DeliverabilityRateLimit # noqa: PLC0415
        d = dict(src_dict)
        adapter = d.pop("adapter")

        badge = d.pop("badge")

        bounce = DeliverabilityRateLimit.from_dict(d.pop("bounce"))




        complaint = DeliverabilityRateLimit.from_dict(d.pop("complaint"))




        name = d.pop("name")

        provider_id = d.pop("providerId")

        status = d.pop("status")

        tone = DeliverabilityProviderHealthTone(d.pop("tone"))




        not_judged = d.pop("notJudged", UNSET)

        _note = d.pop("note", UNSET)
        note: DeliverabilityProviderNote | Unset
        if isinstance(_note,  Unset):
            note = UNSET
        else:
            note = DeliverabilityProviderNote.from_dict(_note)




        deliverability_provider_health = cls(
            adapter=adapter,
            badge=badge,
            bounce=bounce,
            complaint=complaint,
            name=name,
            provider_id=provider_id,
            status=status,
            tone=tone,
            not_judged=not_judged,
            note=note,
        )

        return deliverability_provider_health

