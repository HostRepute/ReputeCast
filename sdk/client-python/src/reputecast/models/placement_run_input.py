from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_run_input_origin import PlacementRunInputOrigin
from ..models.placement_run_input_personalization import PlacementRunInputPersonalization
from ..models.placement_run_input_through import PlacementRunInputThrough
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.placement_content_ref import PlacementContentRef





T = TypeVar("T", bound="PlacementRunInput")



@_attrs_define
class PlacementRunInput:
    """ 
        Attributes:
            content (PlacementContentRef):
            through (PlacementRunInputThrough):
            language (str | Unset):
            origin (PlacementRunInputOrigin | Unset):
            personalization (PlacementRunInputPersonalization | Unset): data: a transactional template's test data set, the
                first one unless testData names one
            providers (list[str] | Unset):
            seeds (list[str] | Unset):
            sender_id (str | Unset):
            test_data (str | Unset):
            variant (str | Unset):
            version (int | Unset): A conditional-content version's number; 0 picks the one with the most recipients
     """

    content: PlacementContentRef
    through: PlacementRunInputThrough
    language: str | Unset = UNSET
    origin: PlacementRunInputOrigin | Unset = UNSET
    personalization: PlacementRunInputPersonalization | Unset = UNSET
    providers: list[str] | Unset = UNSET
    seeds: list[str] | Unset = UNSET
    sender_id: str | Unset = UNSET
    test_data: str | Unset = UNSET
    variant: str | Unset = UNSET
    version: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_content_ref import PlacementContentRef # noqa: PLC0415
        content = self.content.to_dict()

        through = self.through.value

        language = self.language

        origin: str | Unset = UNSET
        if not isinstance(self.origin, Unset):
            origin = self.origin.value


        personalization: str | Unset = UNSET
        if not isinstance(self.personalization, Unset):
            personalization = self.personalization.value


        providers: list[str] | Unset = UNSET
        if not isinstance(self.providers, Unset):
            providers = self.providers



        seeds: list[str] | Unset = UNSET
        if not isinstance(self.seeds, Unset):
            seeds = self.seeds



        sender_id = self.sender_id

        test_data = self.test_data

        variant = self.variant

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "content": content,
            "through": through,
        })
        if language is not UNSET:
            field_dict["language"] = language
        if origin is not UNSET:
            field_dict["origin"] = origin
        if personalization is not UNSET:
            field_dict["personalization"] = personalization
        if providers is not UNSET:
            field_dict["providers"] = providers
        if seeds is not UNSET:
            field_dict["seeds"] = seeds
        if sender_id is not UNSET:
            field_dict["senderId"] = sender_id
        if test_data is not UNSET:
            field_dict["testData"] = test_data
        if variant is not UNSET:
            field_dict["variant"] = variant
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_content_ref import PlacementContentRef # noqa: PLC0415
        d = dict(src_dict)
        content = PlacementContentRef.from_dict(d.pop("content"))




        through = PlacementRunInputThrough(d.pop("through"))




        language = d.pop("language", UNSET)

        _origin = d.pop("origin", UNSET)
        origin: PlacementRunInputOrigin | Unset
        if isinstance(_origin,  Unset):
            origin = UNSET
        else:
            origin = PlacementRunInputOrigin(_origin)




        _personalization = d.pop("personalization", UNSET)
        personalization: PlacementRunInputPersonalization | Unset
        if isinstance(_personalization,  Unset):
            personalization = UNSET
        else:
            personalization = PlacementRunInputPersonalization(_personalization)




        providers = cast(list[str], d.pop("providers", UNSET))


        seeds = cast(list[str], d.pop("seeds", UNSET))


        sender_id = d.pop("senderId", UNSET)

        test_data = d.pop("testData", UNSET)

        variant = d.pop("variant", UNSET)

        version = d.pop("version", UNSET)

        placement_run_input = cls(
            content=content,
            through=through,
            language=language,
            origin=origin,
            personalization=personalization,
            providers=providers,
            seeds=seeds,
            sender_id=sender_id,
            test_data=test_data,
            variant=variant,
            version=version,
        )

        return placement_run_input

